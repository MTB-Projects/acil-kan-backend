package router

import (
	"fmt"

	"acilkan.backend/config"
	"acilkan.backend/internal/auth"
	"acilkan.backend/internal/handler"
	"github.com/gofiber/fiber/v2"
)

const APIVersion = "v1"

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App, cfg *config.Config, userHandler *handler.UserHandler, requestHandler *handler.RequestHandler, hospitalHandler *handler.HospitalHandler) {
	// Health check endpoint (no auth required)
	app.Get("/health", healthCheckHandler)

	// API v1 routes
	v1 := app.Group(fmt.Sprintf("/api/%s", APIVersion))

	// Public routes (no authentication required)
	setupPublicRoutes(v1, requestHandler, hospitalHandler)

	// Protected routes (require Firebase authentication)
	setupProtectedRoutes(v1, cfg, userHandler, requestHandler)
}

// setupPublicRoutes configures public endpoints
func setupPublicRoutes(v1 fiber.Router, requestHandler *handler.RequestHandler, hospitalHandler *handler.HospitalHandler) {
	public := v1.Group("/public")

	// Blood requests - public read access
	public.Get("/requests", requestHandler.GetActiveRequests)

	// Hospital directory for autocomplete (OpenStreetMap data, no personal data)
	public.Get("/hospitals", hospitalHandler.SearchHospitals)
}

// setupProtectedRoutes configures authenticated endpoints
func setupProtectedRoutes(v1 fiber.Router, cfg *config.Config, userHandler *handler.UserHandler, requestHandler *handler.RequestHandler) {
	// Apply authentication middleware to all protected routes
	protected := v1.Group("", auth.FirebaseAuthMiddleware(cfg))

	// User routes
	setupUserRoutes(protected, userHandler)

	// Blood request routes
	setupRequestRoutes(protected, requestHandler)
}

// setupUserRoutes configures user-related endpoints
func setupUserRoutes(protected fiber.Router, userHandler *handler.UserHandler) {
	users := protected.Group("/users")

	users.Get("/me", userHandler.GetProfile)
	users.Put("/me", userHandler.UpdateProfile)
	users.Post("/fcm-token", userHandler.UpdateFCMToken)
}

// setupRequestRoutes configures blood request endpoints
func setupRequestRoutes(protected fiber.Router, requestHandler *handler.RequestHandler) {
	requests := protected.Group("/requests")

	requests.Post("", requestHandler.CreateRequest)
	requests.Get("/my", requestHandler.GetMyRequests)
	requests.Get("/:id", requestHandler.GetRequest)
	requests.Delete("/:id", requestHandler.CancelRequest)
}

// healthCheckHandler returns the health status of the API
func healthCheckHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"version": APIVersion,
	})
}
