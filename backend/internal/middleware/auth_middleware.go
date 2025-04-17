package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/user/minicoinbase/backend/internal/auth"
)

// Protected is a middleware function to verify JWT authentication.
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authorization header"})
		}

		// Expecting "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authorization header format"})
		}

		tokenString := parts[1]
		claims, err := auth.ValidateJWT(tokenString)
		if err != nil {
			// Log the specific error for debugging, but return a generic message
			// log.Printf("JWT validation error: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Store user information in context for downstream handlers
		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)
		// You can add more claims info to locals if needed

		return c.Next()
	}
}
