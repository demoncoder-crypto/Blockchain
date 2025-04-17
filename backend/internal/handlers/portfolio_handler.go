package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/user/minicoinbase/backend/internal/database"
	"github.com/user/minicoinbase/backend/internal/models"
	// TODO: Import ticker package if calculating P&L requires current prices
)

// GetPortfolio retrieves the user's current asset balances.
// TODO: Enhance to calculate P&L based on holdings and current market prices.
func GetPortfolio(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID in token"})
	}

	balances, err := database.GetUserBalances(c.Context(), userID)
	if err != nil {
		log.Printf("Error fetching balances for user %s: %v", userID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve portfolio balances"})
	}

	// If no balances found, return empty array, not null
	if balances == nil {
		balances = make([]*models.Balance, 0)
	}

	// TODO: Calculate portfolio value and P&L
	// 1. Get current market prices (e.g., from ticker.GetCurrentPrices())
	// 2. Iterate through balances
	// 3. For each non-quote asset (e.g., BTC, ETH), calculate its value in the quote currency (e.g., USD)
	//    value = (balance.Available + balance.Locked) * currentPrice[asset+"-USD"]
	// 4. Sum up values + quote currency balance for total portfolio value.
	// 5. P&L calculation requires tracking cost basis (more complex, needs trade history or avg cost)

	// For now, just return the raw balances
	return c.Status(fiber.StatusOK).JSON(balances)
}
