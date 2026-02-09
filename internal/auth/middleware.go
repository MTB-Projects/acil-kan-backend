package auth

import (
	"context"
	"strings"

	"acilkan.backend/config"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// FirebaseAuthMiddleware verifies Firebase ID tokens
// This is the ONLY authentication mechanism - all requests must have valid Firebase token
func FirebaseAuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Unauthorized(c, appErrors.ErrCodeMissingToken, "Authorization header is required")
		}

		// Expected format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return response.Unauthorized(c, appErrors.ErrCodeInvalidToken, "Invalid authorization header format. Expected: Bearer <token>")
		}

		idToken := parts[1]

		// Verify the token using Firebase Admin SDK
		token, err := cfg.FirebaseAuth.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			return response.Unauthorized(c, appErrors.ErrCodeInvalidToken, "Invalid or expired authentication token")
		}

		// Store user UID in context for downstream handlers
		c.Locals("userUID", token.UID)
		c.Locals("userEmail", token.Claims["email"])

		return c.Next()
	}
}

// GetUserUID extracts the authenticated user's UID from context
func GetUserUID(c *fiber.Ctx) string {
	uid, ok := c.Locals("userUID").(string)
	if !ok {
		return ""
	}
	return uid
}

// GetUserEmail extracts the authenticated user's email from context
func GetUserEmail(c *fiber.Ctx) string {
	email, ok := c.Locals("userEmail").(string)
	if !ok {
		return ""
	}
	return email
}
