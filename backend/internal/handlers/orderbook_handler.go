package handlers

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/user/minicoinbase/backend/internal/orderbook"
)

// GetOrderBookDepth retrieves the aggregated depth for a given symbol.
// This endpoint is typically public.
func GetOrderBookDepth(c *fiber.Ctx) error {
	symbol := c.Params("symbol")
	if symbol == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Symbol parameter is required"})
	}
	symbol = strings.ToUpper(symbol)

	// Use the global manager to get the book depth
	depth, err := orderbook.GlobalOrderBookManager.GetBookDepth(symbol)
	if err != nil {
		// This error likely means the manager itself failed, not just an empty book
		log.Printf("Error getting order book depth for symbol %s: %v", symbol, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve order book depth"})
	}

	if depth == nil {
		// Should not happen with GetOrCreateBook logic, but handle defensively
		log.Printf("Nil depth returned for symbol %s", symbol)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve order book depth data"})
	}

	return c.Status(fiber.StatusOK).JSON(depth)
}
