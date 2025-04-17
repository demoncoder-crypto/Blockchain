package main

import (
	"log"

	"github.com/gofiber/contrib/websocket" // Keep original import name
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid" // Need this for type assertion

	// Use module path + directory structure for internal packages
	"github.com/user/minicoinbase/backend/internal/database"
	"github.com/user/minicoinbase/backend/internal/handlers"             // Import handlers
	"github.com/user/minicoinbase/backend/internal/middleware"           // Import middleware
	"github.com/user/minicoinbase/backend/internal/orderbook"            // Import orderbook
	"github.com/user/minicoinbase/backend/internal/ticker"               // Import ticker
	internalws "github.com/user/minicoinbase/backend/internal/websocket" // Alias internal websocket
)

func main() {
	// Initialize Database
	database.InitDB()
	defer database.CloseDB() // Ensure DB connection is closed on exit

	// Initialize WebSocket Hub
	internalws.InitializeGlobalHub() // Use alias

	// Initialize Price Ticker (starts broadcasting to the hub)
	ticker.InitTicker()

	// Initialize Order Book Manager
	orderbook.InitManager()

	app := fiber.New()

	// --- WebSocket Routes ---
	// Needs to be defined before the /api group if it shouldn't inherit middleware
	wsGroup := app.Group("/ws")
	wsGroup.Use("/", func(c *fiber.Ctx) error {
		// Middleware to check for upgrade request
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	// Price feed WebSocket endpoint - Use websocket.New
	wsGroup.Get("/prices", websocket.New(handlers.PriceWSEndpoint))

	// --- API Routes ---
	api := app.Group("/api") // Group routes under /api

	// Health check (Public)
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("Mini-Coinbase API is healthy!")
	})

	// Order Book Depth (Public)
	api.Get("/book/:symbol", handlers.GetOrderBookDepth)

	// Auth routes (Public)
	authGroup := api.Group("/auth")
	authGroup.Post("/signup", handlers.Signup)
	authGroup.Post("/login", handlers.Login)

	// --- Protected Routes ---
	// Apply the Protected middleware to all routes defined after this
	api.Use(middleware.Protected())

	// Example Protected Route: Get current user info
	api.Get("/me", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		username, ok2 := c.Locals("username").(string)

		if !ok || !ok2 {
			// This shouldn't happen if middleware ran correctly, but good practice to check
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get user info from context"})
		}

		return c.JSON(fiber.Map{
			"message":  "Successfully authenticated",
			"user_id":  userID,
			"username": username,
		})
	})

	// Order Routes (Protected)
	ordersGroup := api.Group("/orders")
	ordersGroup.Post("/", handlers.CreateOrder)
	ordersGroup.Get("/", handlers.GetOrders)         // Get user's orders
	ordersGroup.Get("/:id", handlers.GetOrderByID)   // Get specific order by ID
	ordersGroup.Delete("/:id", handlers.CancelOrder) // Cancel specific order by ID

	// Portfolio Route (Protected)
	api.Get("/portfolio", handlers.GetPortfolio)

	// TODO: Add other PROTECTED routes here (e.g., Trade History?)

	log.Println("Starting server on :8080")
	log.Fatal(app.Listen(":8080"))
}
