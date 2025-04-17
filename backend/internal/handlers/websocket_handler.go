package handlers

import (
	"log"

	"github.com/gofiber/contrib/websocket"
	ws "github.com/user/minicoinbase/backend/internal/websocket" // Alias websocket package
)

// PriceWSEndpoint is the handler for the WebSocket price feed.
func PriceWSEndpoint(c *websocket.Conn) {
	// c.Locals is fiber.Ctx specific, Conn doesn't have direct access.
	// If you need authentication for WS, it needs to be handled differently,
	// often via a token passed in the connection URL or an initial message.
	// For now, we assume public access to the price feed.

	client := &ws.Client{
		Conn: c,
		Send: make(chan []byte, 256), // Buffered channel for outgoing messages to this client
	}

	// Register the client with the hub
	ws.GlobalHub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.

	// Goroutine to handle writing messages from the hub to the client
	go clientWritePump(client)

	// Goroutine to handle reading messages from the client (e.g., ping/pong, subscriptions)
	go clientReadPump(client)

	log.Printf("WebSocket connection established: %s", c.RemoteAddr())
	// The handler function returns here, but the goroutines keep running.
}

// clientWritePump pumps messages from the hub to the websocket connection.
func clientWritePump(client *ws.Client) {
	defer func() {
		// Ensure connection is closed on exit
		client.Conn.Close()
		log.Printf("Write pump stopped for %s", client.Conn.RemoteAddr())
	}()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error writing message to %s: %v", client.Conn.RemoteAddr(), err)
			// If write fails, assume client disconnected
			ws.GlobalHub.Unregister <- client
			return
		}
	}
	// If client.Send channel is closed by the hub, this loop terminates
}

// clientReadPump pumps messages from the websocket connection to the hub (or handles them).
// Currently, it just handles disconnects and ping/pong.
func clientReadPump(client *ws.Client) {
	defer func() {
		// When this function exits (e.g., client disconnects), unregister the client
		ws.GlobalHub.Unregister <- client
		client.Conn.Close()
		log.Printf("Read pump stopped for %s", client.Conn.RemoteAddr())
	}()

	// Configure connection properties (optional)
	// client.Conn.SetReadLimit(maxMessageSize)
	// client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	// client.Conn.SetPongHandler(func(string) error { client.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		// ReadMessage blocks until a message is received or an error occurs
		messageType, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client disconnected unexpectedly %s: %v", client.Conn.RemoteAddr(), err)
			} else {
				log.Printf("Error reading message from %s: %v", client.Conn.RemoteAddr(), err)
			}
			break // Exit loop on error
		}

		// Process received message (optional)
		// Currently, we don't expect messages from the client for the price feed,
		// but you could handle subscription messages here.
		log.Printf("Received message type %d from %s: %s", messageType, client.Conn.RemoteAddr(), message)
	}
}
