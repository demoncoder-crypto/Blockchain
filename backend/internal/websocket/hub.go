package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/user/minicoinbase/backend/internal/ticker"
)

// Client represents a single WebSocket client connection.
type Client struct {
	Conn *websocket.Conn
	Send chan []byte // Buffered channel for outbound messages
}

// Hub manages WebSocket clients and broadcasts messages.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte  // Keep this unexported if only used internally
	Register   chan *Client // Exported
	Unregister chan *Client // Exported
	mu         sync.RWMutex
}

var GlobalHub *Hub

// NewHub creates and initializes a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client), // Use exported name
		Unregister: make(chan *Client), // Use exported name
	}
}

// Run starts the Hub's event loop.
func (h *Hub) Run() {
	log.Println("Starting WebSocket Hub...")
	// Start listening to the price ticker updates
	go h.listenToPriceUpdates()

	for {
		select {
		case client := <-h.Register: // Use exported name
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %s", client.Conn.RemoteAddr())
			// Maybe send initial data (e.g., current prices) upon registration
			// currentPrices := ticker.GetCurrentPrices()
			// msg, _ := json.Marshal(currentPrices)
			// client.Send <- msg

		case client := <-h.Unregister: // Use exported name
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				log.Printf("Client unregistered: %s", client.Conn.RemoteAddr())
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			// Send message to all registered clients
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client's send buffer is full, close connection
					log.Printf("Client send buffer full, closing connection: %s", client.Conn.RemoteAddr())
					close(client.Send)
					delete(h.clients, client) // Need write lock for this, potential improvement needed
				}
			}
			h.mu.RUnlock()
		}
	}
}

// listenToPriceUpdates listens to the ticker's PriceUpdates channel and broadcasts them.
func (h *Hub) listenToPriceUpdates() {
	log.Println("Hub listening for price updates...")
	for update := range ticker.PriceUpdates {
		// Marshal the update to JSON
		msgBytes, err := json.Marshal(update)
		if err != nil {
			log.Printf("Error marshalling price update: %v", err)
			continue
		}
		// Send JSON to the broadcast channel
		h.broadcast <- msgBytes
	}
}

// InitializeGlobalHub creates and runs the global Hub instance.
func InitializeGlobalHub() {
	GlobalHub = NewHub()
	go GlobalHub.Run()
}
