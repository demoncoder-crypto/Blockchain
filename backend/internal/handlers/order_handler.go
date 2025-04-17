package handlers

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/user/minicoinbase/backend/internal/database"
	"github.com/user/minicoinbase/backend/internal/models"
	"github.com/user/minicoinbase/backend/internal/orderbook" // Import orderbook
	// TODO: Import orderbook package when created
)

// CreateOrderRequest defines the expected JSON body for creating an order
type CreateOrderRequest struct {
	Symbol   string  `json:"symbol"`   // e.g., "BTC-USD"
	Type     string  `json:"type"`     // e.g., "limit", "market"
	Side     string  `json:"side"`     // e.g., "buy", "sell"
	Price    float64 `json:"price"`    // Required for limit orders
	Quantity float64 `json:"quantity"` // Amount of base asset (e.g., BTC)
}

// CreateOrder handles the creation of new trading orders.
func CreateOrder(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID in token"})
	}

	req := new(CreateOrderRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse request body"})
	}

	// --- Basic Validation ---
	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	req.Side = strings.ToLower(strings.TrimSpace(req.Side))

	if req.Symbol == "" || req.Quantity <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Symbol and positive quantity are required"})
	}
	parts := strings.Split(req.Symbol, "-")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid symbol format, expected BASE-QUOTE"})
	}
	baseAsset := parts[0]
	quoteAsset := parts[1]

	if req.Side != "buy" && req.Side != "sell" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid side, must be 'buy' or 'sell'"})
	}
	if req.Type != "limit" && req.Type != "market" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid type, must be 'limit' or 'market'"})
	}
	if req.Type == "limit" && req.Price <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Positive price is required for limit orders"})
	}
	// TODO: Add more validation (precision, allowed symbols?)

	order := &models.Order{
		UserID:   userID,
		Symbol:   req.Symbol,
		Type:     req.Type,
		Side:     req.Side,
		Quantity: req.Quantity,
		Status:   "open", // Will be created with this status if validation/locking succeeds
	}
	if req.Type == "limit" {
		order.Price = req.Price
	}

	// --- Transactional Logic ---
	tx, err := database.DB.Begin(c.Context())
	if err != nil {
		log.Printf("Failed to begin transaction for user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error starting transaction"})
	}
	// Ensure rollback happens if anything goes wrong before commit
	defer tx.Rollback(c.Context())

	// 1. Check and Lock Funds
	var lockAsset string
	var lockAmount float64

	if req.Side == "buy" {
		lockAsset = quoteAsset
		if req.Type == "limit" {
			lockAmount = req.Price * req.Quantity
		} else { // Market Buy
			// TODO: Implement market order cost estimation & locking
			// This is complex: need current market price, potential slippage buffer.
			// For now, reject market buys.
			log.Printf("Market buy orders not yet supported (user %s)", userID)
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "Market buy orders are not yet supported"})
		}
	} else { // Sell side
		lockAsset = baseAsset
		lockAmount = req.Quantity
	}

	// Ensure the balance exists before trying to lock (avoids confusing errors)
	_, err = database.GetOrCreateBalanceInTx(c.Context(), tx, userID, lockAsset)
	if err != nil {
		log.Printf("Failed to get/create %s balance for user %s in tx: %v", lockAsset, userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Database error accessing %s balance", lockAsset)})
	}

	// Attempt to lock the required funds
	err = database.LockFunds(c.Context(), tx, userID, lockAsset, lockAmount)
	if err != nil {
		log.Printf("Failed to lock %f %s for user %s order: %v", lockAmount, lockAsset, userID, err)
		// Return a user-friendly insufficient funds error or the specific lock error
		userMsg := fmt.Sprintf("Failed to lock funds: %s", err.Error())
		if strings.Contains(err.Error(), "insufficient funds") { // Make error more generic for client
			userMsg = fmt.Sprintf("Insufficient %s balance to place order", lockAsset)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": userMsg})
	}
	log.Printf("Successfully locked %f %s for user %s", lockAmount, lockAsset, userID)

	// 2. Create Order Record
	if err := database.CreateOrder(c.Context(), tx, order); err != nil {
		log.Printf("Error creating order in DB for user %s (after locking funds): %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save order after locking funds"})
	}

	// 3. Commit Transaction
	if err := tx.Commit(c.Context()); err != nil {
		log.Printf("Failed to commit transaction for user %s order %s: %v", userID, order.ID, err)
		// Attempted to lock funds and create order, but commit failed. Funds are rolled back.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error finalizing order"})
	}

	// Transaction successful!
	log.Printf("Order %s created and funds locked successfully for user %s", order.ID, userID)

	// Submit order to matching engine/order book AFTER successful commit
	if err := orderbook.GlobalOrderBookManager.SubmitOrder(order); err != nil {
		// Log error, but don't necessarily fail the HTTP request as the order IS in the DB.
		// This indicates an issue submitting to the live matching engine.
		log.Printf("CRITICAL: Failed to submit committed order %s to order book: %v", order.ID, err)
		// Maybe return a specific status or message indicating this?
	}

	return c.Status(fiber.StatusCreated).JSON(order)
}

// GetOrders retrieves the list of active orders for the authenticated user.
func GetOrders(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID in token"})
	}

	orders, err := database.GetUserOrders(c.Context(), userID)
	if err != nil {
		log.Printf("Error fetching orders for user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve orders"})
	}

	return c.Status(fiber.StatusOK).JSON(orders)
}

// GetOrderByID retrieves a specific order by its ID.
func GetOrderByID(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID in token"})
	}

	orderIDParam := c.Params("id")
	orderID, err := uuid.Parse(orderIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID format"})
	}

	order, err := database.GetOrderByID(c.Context(), orderID)
	if err != nil {
		log.Printf("Error fetching order %s: %v", orderID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve order details"})
	}

	if order == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Order not found"})
	}

	// Ensure the user owns this order
	if order.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You do not have permission to view this order"})
	}

	return c.Status(fiber.StatusOK).JSON(order)
}

// CancelOrder handles the cancellation of an existing order.
func CancelOrder(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID in token"})
	}

	orderIDParam := c.Params("id")
	orderID, err := uuid.Parse(orderIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID format"})
	}

	// --- Transactional Logic ---
	tx, err := database.DB.Begin(c.Context())
	if err != nil {
		log.Printf("CancelOrder: Failed to begin transaction for user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error starting transaction"})
	}
	defer tx.Rollback(c.Context())

	// 1. Attempt to cancel the order in the DB (locks row, checks ownership & status)
	originalOrder, err := database.CancelOrder(c.Context(), tx, userID, orderID)
	if err != nil {
		log.Printf("CancelOrder: Failed for user %s, order %s: %v", userID, orderID, err)
		userMsg := err.Error()
		status := fiber.StatusInternalServerError
		if strings.Contains(userMsg, "not found or permission denied") {
			status = fiber.StatusNotFound // Or StatusForbidden depending on desired behavior
			userMsg = "Order not found or you do not have permission to cancel it"
		} else if strings.Contains(userMsg, "not in a cancellable state") {
			status = fiber.StatusBadRequest
		} else {
			userMsg = "Failed to cancel order"
		}
		return c.Status(status).JSON(fiber.Map{"error": userMsg})
	}

	// 2. Determine which funds to unlock
	parts := strings.Split(originalOrder.Symbol, "-")
	baseAsset := parts[0]
	quoteAsset := parts[1]
	var unlockAsset string
	var unlockAmount float64

	if originalOrder.Side == "buy" {
		unlockAsset = quoteAsset
		if originalOrder.Type == "limit" {
			unlockAmount = originalOrder.Price * originalOrder.Quantity
		} else {
			// Market buy cancellation logic if market buys were supported
			log.Printf("CancelOrder: Market buy cancellation logic needed user %s, order %s", userID, orderID)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot cancel market buy order (logic pending)"})
		}
	} else { // Sell side
		unlockAsset = baseAsset
		unlockAmount = originalOrder.Quantity
	}

	// 3. Unlock the previously locked funds
	if err := database.UnlockFunds(c.Context(), tx, userID, unlockAsset, unlockAmount); err != nil {
		log.Printf("CancelOrder: CRITICAL: Failed to unlock %f %s for user %s, order %s after status update: %v",
			unlockAmount, unlockAsset, userID, orderID, err)
		// Order status is 'cancelled', but funds might still be locked! Requires manual intervention.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Order cancelled, but failed to unlock funds. Please contact support."}) // Critical error
	}
	log.Printf("CancelOrder: Unlocked %f %s for user %s, order %s", unlockAmount, unlockAsset, userID, orderID)

	// 4. Commit Transaction
	if err := tx.Commit(c.Context()); err != nil {
		log.Printf("CancelOrder: Failed to commit transaction for user %s order %s: %v", userID, orderID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error finalizing order cancellation"})
	}

	// Transaction successful!
	log.Printf("Order %s cancelled successfully in DB for user %s", orderID, userID)

	// Notify order book/matching engine AFTER successful commit
	if err := orderbook.GlobalOrderBookManager.CancelOrder(originalOrder); err != nil {
		// Order is cancelled in DB, but failed to remove from live book. Log critically.
		log.Printf("CRITICAL: Failed to cancel order %s from order book after DB commit: %v", originalOrder.ID, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Order cancelled successfully"})
}
