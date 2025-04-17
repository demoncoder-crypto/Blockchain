package handlers

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/user/minicoinbase/backend/internal/auth"
	"github.com/user/minicoinbase/backend/internal/database"
	"github.com/user/minicoinbase/backend/internal/models"
)

// SignupRequest defines the expected JSON body for signup
type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest defines the expected JSON body for login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse defines the JSON response for successful auth
type AuthResponse struct {
	Token    string       `json:"token"`
	User     *models.User `json:"user"` // Return basic user info (excluding password hash)
	IssuedAt time.Time    `json:"issued_at"`
}

// Signup handles user registration.
func Signup(c *fiber.Ctx) error {
	req := new(SignupRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse request body"})
	}

	// Basic validation
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username and password cannot be empty"})
	}
	// TODO: Add more robust validation (e.g., password complexity, username format)

	// Check if user already exists
	existingUser, err := database.GetUserByUsername(c.Context(), req.Username)
	if err != nil {
		log.Printf("Error checking username %s: %v", req.Username, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error checking username"})
	}
	if existingUser != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Username already taken"})
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password for %s: %v", req.Username, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to process password"})
	}

	// Create user in database
	newUser, err := database.CreateUser(c.Context(), req.Username, hashedPassword)
	if err != nil {
		// TODO: Handle specific DB errors like unique constraint violation potentially missed by first check
		log.Printf("Error creating user %s: %v", req.Username, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user"})
	}

	// Generate JWT
	token, err := auth.GenerateJWT(newUser.ID, newUser.Username)
	if err != nil {
		log.Printf("Error generating JWT for user %s: %v", newUser.Username, err)
		// User was created, but token failed - problematic state. Log carefully.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "User created, but failed to generate token"})
	}

	// Don't send password hash back
	newUser.Password = ""

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		Token:    token,
		User:     newUser,
		IssuedAt: time.Now(),
	})
}

// Login handles user authentication.
func Login(c *fiber.Ctx) error {
	req := new(LoginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse request body"})
	}

	// Basic validation
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username and password cannot be empty"})
	}

	// Find user by username
	user, err := database.GetUserByUsername(c.Context(), req.Username)
	if err != nil {
		log.Printf("Error finding user %s: %v", req.Username, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error finding user"})
	}
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password"})
	}

	// Check password
	if !auth.CheckPasswordHash(req.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password"})
	}

	// Generate JWT
	token, err := auth.GenerateJWT(user.ID, user.Username)
	if err != nil {
		log.Printf("Error generating JWT for user %s: %v", user.Username, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate token"})
	}

	// Don't send password hash back
	user.Password = ""

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Token:    token,
		User:     user,
		IssuedAt: time.Now(),
	})
}
