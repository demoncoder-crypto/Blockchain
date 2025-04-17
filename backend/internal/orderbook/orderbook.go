package orderbook

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/minicoinbase/backend/internal/models"
)

// Limit represents a price level in the order book.
// Contains a list of orders at that price.
// For simplicity now, we might just store orders directly in sorted slices.

// Order represents an order within the order book.
// We might reuse models.Order or have a simplified internal representation.
// For now, using models.Order.

// OrderBookSide represents either the bid or ask side of the book.
// Using sorted slices for simplicity.
// Bids should be sorted high to low price.
// Asks should be sorted low to high price.

// OrderBook represents the order book for a single trading pair.
type OrderBook struct {
	symbol string
	mu     sync.RWMutex
	// Using simple slices and sorting for now.
	// For performance, consider using heaps or balanced trees.
	Bids []*models.Order // Sorted descending by price
	Asks []*models.Order // Sorted ascending by price

	// Optional: Map for quick order lookup by ID for cancellation
	Orders map[uuid.UUID]*models.Order
}

// NewOrderBook creates a new order book for a given symbol.
func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		symbol: symbol,
		Bids:   make([]*models.Order, 0),
		Asks:   make([]*models.Order, 0),
		Orders: make(map[uuid.UUID]*models.Order),
	}
}

// AddOrder adds a new order to the book and triggers matching.
// Returns a list of trades executed.
func (ob *OrderBook) AddOrder(order *models.Order) ([]*Trade, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	// Basic validation (ensure correct symbol, type)
	if order.Symbol != ob.symbol {
		return nil, fmt.Errorf("order symbol %s does not match book symbol %s", order.Symbol, ob.symbol)
	}
	if order.Type != "limit" {
		// Only limit orders can rest on the book
		// TODO: Handle market orders - they would match immediately without resting.
		return nil, fmt.Errorf("only limit orders can be added directly to the book")
	}

	// Check if order already exists (e.g., resubmission attempt?)
	if _, exists := ob.Orders[order.ID]; exists {
		return nil, fmt.Errorf("order %s already exists in the book", order.ID)
	}

	// Add to lookup map
	ob.Orders[order.ID] = order

	// TODO: Implement Matching Logic Here
	trades := ob.matchOrder(order)

	// If the order is not fully filled, add the remainder to the book
	if order.Quantity > 0 { // Assuming Quantity represents remaining quantity
		if order.Side == "buy" {
			ob.addBid(order)
		} else {
			ob.addAsk(order)
		}
	}

	// TODO: Update order status (e.g., partially_filled, filled) based on trades
	// This should likely happen outside the order book, maybe in a service layer
	// that calls the DB updates after getting trades from the book.

	return trades, nil
}

// matchOrder attempts to match the incoming order against the resting orders.
// Modifies the incoming order's quantity and returns executed trades.
// NOTE: This is a simplified placeholder implementation.
func (ob *OrderBook) matchOrder(incomingOrder *models.Order) []*Trade {
	trades := make([]*Trade, 0)
	if incomingOrder.Side == "buy" {
		// Match against asks (lowest price first)
		for i := 0; i < len(ob.Asks) && incomingOrder.Quantity > 0; {
			ask := ob.Asks[i]
			if incomingOrder.Price >= ask.Price { // Match possible
				matchQuantity := math.Min(incomingOrder.Quantity, ask.Quantity)
				trade := &Trade{
					TakerOrderID: incomingOrder.ID,
					MakerOrderID: ask.ID,
					Symbol:       ob.symbol,
					Price:        ask.Price, // Trade occurs at the resting order's price
					Quantity:     matchQuantity,
					Timestamp:    time.Now(),
				}
				trades = append(trades, trade)

				incomingOrder.Quantity -= matchQuantity
				ask.Quantity -= matchQuantity

				if ask.Quantity == 0 {
					// Remove filled ask order
					delete(ob.Orders, ask.ID)
					ob.Asks = append(ob.Asks[:i], ob.Asks[i+1:]...)
					// Don't increment i, the next element is now at index i
				} else {
					i++ // Move to next ask
				}
			} else {
				// Incoming bid price is lower than the best ask, no more matches
				break
			}
		}
	} else { // Incoming order is a sell
		// Match against bids (highest price first)
		for i := 0; i < len(ob.Bids) && incomingOrder.Quantity > 0; {
			bid := ob.Bids[i]
			if incomingOrder.Price <= bid.Price { // Match possible
				matchQuantity := math.Min(incomingOrder.Quantity, bid.Quantity)
				trade := &Trade{
					TakerOrderID: incomingOrder.ID,
					MakerOrderID: bid.ID,
					Symbol:       ob.symbol,
					Price:        bid.Price, // Trade occurs at the resting order's price
					Quantity:     matchQuantity,
					Timestamp:    time.Now(),
				}
				trades = append(trades, trade)

				incomingOrder.Quantity -= matchQuantity
				bid.Quantity -= matchQuantity

				if bid.Quantity == 0 {
					// Remove filled bid order
					delete(ob.Orders, bid.ID)
					ob.Bids = append(ob.Bids[:i], ob.Bids[i+1:]...)
					// Don't increment i
				} else {
					i++ // Move to next bid
				}
			} else {
				// Incoming ask price is higher than the best bid, no more matches
				break
			}
		}
	}
	return trades
}

// addBid inserts a bid order into the sorted Bids slice.
func (ob *OrderBook) addBid(order *models.Order) {
	// Find insertion point to maintain sort order (descending price)
	i := sort.Search(len(ob.Bids), func(j int) bool { return ob.Bids[j].Price <= order.Price })
	ob.Bids = append(ob.Bids, nil)   // Make space
	copy(ob.Bids[i+1:], ob.Bids[i:]) // Shift elements right
	ob.Bids[i] = order               // Insert
}

// addAsk inserts an ask order into the sorted Asks slice.
func (ob *OrderBook) addAsk(order *models.Order) {
	// Find insertion point to maintain sort order (ascending price)
	i := sort.Search(len(ob.Asks), func(j int) bool { return ob.Asks[j].Price >= order.Price })
	ob.Asks = append(ob.Asks, nil)   // Make space
	copy(ob.Asks[i+1:], ob.Asks[i:]) // Shift elements right
	ob.Asks[i] = order               // Insert
}

// CancelOrder removes an order from the book.
func (ob *OrderBook) CancelOrder(orderID uuid.UUID) (*models.Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.Orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found in book", orderID)
	}

	// Remove from lookup map
	delete(ob.Orders, orderID)

	// Remove from Bids or Asks slice
	if order.Side == "buy" {
		for i, bid := range ob.Bids {
			if bid.ID == orderID {
				ob.Bids = append(ob.Bids[:i], ob.Bids[i+1:]...)
				break
			}
		}
	} else {
		for i, ask := range ob.Asks {
			if ask.ID == orderID {
				ob.Asks = append(ob.Asks[:i], ob.Asks[i+1:]...)
				break
			}
		}
	}

	return order, nil
}

// GetDepth returns a snapshot of the order book depth (e.g., top N levels).
type BookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

type OrderBookDepth struct {
	Symbol string      `json:"symbol"`
	Bids   []BookLevel `json:"bids"` // Aggregated bids [price, total_quantity]
	Asks   []BookLevel `json:"asks"` // Aggregated asks [price, total_quantity]
}

// GetDepth aggregates quantities at each price level.
func (ob *OrderBook) GetDepth() *OrderBookDepth {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	depth := &OrderBookDepth{
		Symbol: ob.symbol,
		Bids:   make([]BookLevel, 0),
		Asks:   make([]BookLevel, 0),
	}

	// Aggregate Bids (already sorted high to low)
	levelMapBids := make(map[float64]float64)
	for _, bid := range ob.Bids {
		levelMapBids[bid.Price] += bid.Quantity
	}
	// Convert map to sorted slice
	bidsPrices := make([]float64, 0, len(levelMapBids))
	for price := range levelMapBids {
		bidsPrices = append(bidsPrices, price)
	}
	sort.Slice(bidsPrices, func(i, j int) bool { return bidsPrices[i] > bidsPrices[j] }) // Descending
	for _, price := range bidsPrices {
		depth.Bids = append(depth.Bids, BookLevel{Price: price, Quantity: levelMapBids[price]})
	}

	// Aggregate Asks (already sorted low to high)
	levelMapAsks := make(map[float64]float64)
	for _, ask := range ob.Asks {
		levelMapAsks[ask.Price] += ask.Quantity
	}
	// Convert map to sorted slice
	asksPrices := make([]float64, 0, len(levelMapAsks))
	for price := range levelMapAsks {
		asksPrices = append(asksPrices, price)
	}
	sort.Slice(asksPrices, func(i, j int) bool { return asksPrices[i] < asksPrices[j] }) // Ascending
	for _, price := range asksPrices {
		depth.Asks = append(depth.Asks, BookLevel{Price: price, Quantity: levelMapAsks[price]})
	}

	// Optional: Limit depth to top N levels
	// const maxDepthLevels = 20
	// if len(depth.Bids) > maxDepthLevels { depth.Bids = depth.Bids[:maxDepthLevels] }
	// if len(depth.Asks) > maxDepthLevels { depth.Asks = depth.Asks[:maxDepthLevels] }

	return depth
}

// Trade represents a successfully matched trade.
type Trade struct {
	TakerOrderID uuid.UUID `json:"taker_order_id"`
	MakerOrderID uuid.UUID `json:"maker_order_id"`
	Symbol       string    `json:"symbol"`
	Price        float64   `json:"price"`
	Quantity     float64   `json:"quantity"`
	Timestamp    time.Time `json:"timestamp"`
}
