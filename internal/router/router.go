package router

import (
	"fmt"

	"acilkan.backend/config"
	"acilkan.backend/internal/auth"
	"acilkan.backend/internal/handler"
	"acilkan.backend/internal/model"
	"acilkan.backend/internal/repository"
	"github.com/gofiber/fiber/v2"
)

const APIVersion = "v1"

// Handlers groups everything the routes need
type Handlers struct {
	User        *handler.UserHandler
	Request     *handler.RequestHandler
	Hospital    *handler.HospitalHandler
	Institution *handler.InstitutionHandler
	// Users is used by the admin role check
	Users *repository.UserRepository
}

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App, cfg *config.Config, h Handlers) {
	// Health check endpoint (no auth required)
	app.Get("/health", healthCheckHandler)

	// API v1 routes
	v1 := app.Group(fmt.Sprintf("/api/%s", APIVersion))

	// Public routes (no authentication required)
	public := v1.Group("/public")
	// Blood requests - public read access, no personal data
	public.Get("/requests", h.Request.GetActiveRequests)
	// Hospital directory for autocomplete (OpenStreetMap data, no personal data)
	public.Get("/hospitals", h.Hospital.SearchHospitals)

	// Protected routes (require Firebase authentication)
	protected := v1.Group("", auth.FirebaseAuthMiddleware(cfg))

	users := protected.Group("/users")
	users.Get("/me", h.User.GetProfile)
	users.Put("/me", h.User.UpdateProfile)
	users.Post("/fcm-token", h.User.UpdateFCMToken)

	requests := protected.Group("/requests")
	requests.Post("", h.Request.CreateRequest)
	requests.Get("/my", h.Request.GetMyRequests)
	requests.Get("/:id", h.Request.GetRequest)
	requests.Delete("/:id", h.Request.CancelRequest)

	// Institution verification: users apply, admins review
	institution := protected.Group("/institution")
	institution.Post("/applications", h.Institution.Apply)
	institution.Get("/applications/me", h.Institution.MyApplication)

	admin := protected.Group("/admin", auth.RequireRole(h.Users, model.RoleAdmin))
	admin.Get("/institution-applications", h.Institution.List)
	admin.Post("/institution-applications/:id/approve", h.Institution.Approve)
	admin.Post("/institution-applications/:id/reject", h.Institution.Reject)
}

// healthCheckHandler returns the health status of the API
func healthCheckHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"version": APIVersion,
	})
}
