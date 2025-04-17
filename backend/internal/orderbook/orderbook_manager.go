package orderbook

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/user/minicoinbase/backend/internal/models"
	// TODO: Import database package for trade processing?
)

// Manager holds and manages multiple OrderBook instances.
type Manager struct {
	mu    sync.RWMutex
	books map[string]*OrderBook // Key: symbol (e.g., "BTC-USD")
	// TODO: Add channel for broadcasting trades?
}

var GlobalOrderBookManager *Manager

// InitManager initializes the global order book manager.
func InitManager() {
	log.Println("Initializing Order Book Manager...")
	GlobalOrderBookManager = &Manager{
		books: make(map[string]*OrderBook),
	}
	// TODO: Pre-create books for known symbols?
	// GlobalOrderBookManager.GetOrCreateBook("BTC-USD")
	// GlobalOrderBookManager.GetOrCreateBook("ETH-USD")
	// GlobalOrderBookManager.GetOrCreateBook("SOL-USD")
}

// GetOrCreateBook retrieves an existing order book or creates a new one for the symbol.
func (m *Manager) GetOrCreateBook(symbol string) *OrderBook {
	symbol = strings.ToUpper(symbol)
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if exists {
		return book
	}

	// Doesn't exist, need write lock to create
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check in case it was created between RUnlock and Lock
	book, exists = m.books[symbol]
	if exists {
		return book
	}

	// Create new book
	log.Printf("Creating new order book for symbol: %s", symbol)
	newBook := NewOrderBook(symbol)
	m.books[symbol] = newBook
	return newBook
}

// SubmitOrder adds an order to the appropriate book and handles resulting trades.
func (m *Manager) SubmitOrder(order *models.Order) error {
	book := m.GetOrCreateBook(order.Symbol)
	trades, err := book.AddOrder(order)
	if err != nil {
		log.Printf("Error adding order %s to book %s: %v", order.ID, order.Symbol, err)
		return err
	}

	if len(trades) > 0 {
		log.Printf("Order %s generated %d trades on book %s", order.ID, len(trades), order.Symbol)
		// TODO: Process Trades!
		// - Start DB transaction
		// - Update maker order status/quantity in DB
		// - Update taker order status/quantity in DB
		// - Update balances for both maker and taker users (using database.UpdateBalancesForFill)
		// - Record the trade itself in a separate trades table?
		// - Commit DB transaction
		// - Broadcast trade event (e.g., via WebSocket)?
		go m.processTrades(trades) // Process trades asynchronously for now
	}

	return nil
}

// CancelOrder removes an order from the appropriate book.
func (m *Manager) CancelOrder(order *models.Order) error {
	book := m.GetOrCreateBook(order.Symbol) // Book should exist if order was placed
	_, err := book.CancelOrder(order.ID)
	if err != nil {
		log.Printf("Error cancelling order %s from book %s: %v", order.ID, order.Symbol, err)
		return err
	}
	log.Printf("Order %s cancelled from book %s", order.ID, order.Symbol)
	return nil
}

// GetBookDepth returns the depth for a specific symbol.
func (m *Manager) GetBookDepth(symbol string) (*OrderBookDepth, error) {
	symbol = strings.ToUpper(symbol)
	book := m.GetOrCreateBook(symbol) // Get or create (might be empty if no orders yet)
	if book == nil {
		// This shouldn't happen with GetOrCreateBook logic
		return nil, fmt.Errorf("failed to get order book for symbol %s", symbol)
	}
	return book.GetDepth(), nil
}

// processTrades (placeholder) handles database updates after trades occur.
func (m *Manager) processTrades(trades []*Trade) {
	log.Printf("Processing %d trades...", len(trades))
	// !!! This needs full implementation with database transactions !!!
	for _, trade := range trades {
		log.Printf(" Trade: Maker=%s, Taker=%s, Qty=%f, Price=%f, Time=%s",
			trade.MakerOrderID, trade.TakerOrderID, trade.Quantity, trade.Price, trade.Timestamp)

		// TODO:
		// 1. Get maker & taker order details from DB (need UserID)
		// 2. Begin transaction
		// 3. Update maker order status/quantity (filled/partially_filled)
		// 4. Update taker order status/quantity
		// 5. Update maker balance (e.g., using UpdateBalancesForFill)
		// 6. Update taker balance
		// 7. Commit
	}
	log.Printf("Finished processing %d trades (placeholder).", len(trades))
}
