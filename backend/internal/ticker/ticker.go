package ticker

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

// PriceUpdate represents a single price update for a symbol.
type PriceUpdate struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Ts     int64   `json:"ts"` // Unix timestamp milliseconds
}

var (
	currentPrices = make(map[string]float64)
	mu            sync.RWMutex
	// Channel to broadcast price updates
	PriceUpdates = make(chan PriceUpdate, 100) // Buffered channel
	symbols      = []string{"BTC-USD", "ETH-USD", "SOL-USD"}
)

// InitTicker starts the background process to simulate price changes.
func InitTicker() {
	mu.Lock()
	// Initialize starting prices
	currentPrices["BTC-USD"] = 60000.00
	currentPrices["ETH-USD"] = 3000.00
	currentPrices["SOL-USD"] = 150.00
	mu.Unlock()

	log.Println("Initializing price ticker...")
	go runTicker()
}

// runTicker periodically updates prices and broadcasts them.
func runTicker() {
	ticker := time.NewTicker(2 * time.Second) // Update prices every 2 seconds
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		for _, symbol := range symbols {
			// Simulate a small price change (+/- 0.5%)
			oldPrice := currentPrices[symbol]
			changePercent := (rand.Float64() - 0.5) / 100 // Max 0.5% change up or down
			newPrice := oldPrice * (1 + changePercent)
			// Ensure price doesn't go negative (unlikely but possible with large swings)
			if newPrice < 0 {
				newPrice = oldPrice * 0.1 // drastic recovery if negative
			}
			currentPrices[symbol] = newPrice

			// Create and send update
			update := PriceUpdate{
				Symbol: symbol,
				Price:  newPrice,
				Ts:     time.Now().UnixMilli(),
			}

			// Non-blocking send to avoid blocking ticker if channel is full
			select {
			case PriceUpdates <- update:
			default:
				log.Println("Price update channel full, dropping update for", symbol)
			}
		}
		mu.Unlock()
	}
}

// GetCurrentPrices returns a copy of the current prices.
func GetCurrentPrices() map[string]float64 {
	mu.RLock()
	defer mu.RUnlock()
	// Return a copy to avoid race conditions on the caller's side
	pricesCopy := make(map[string]float64, len(currentPrices))
	for k, v := range currentPrices {
		pricesCopy[k] = v
	}
	return pricesCopy
}
