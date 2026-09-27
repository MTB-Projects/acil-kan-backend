package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"acilkan.backend/config"
	"acilkan.backend/internal/handler"
	"acilkan.backend/internal/notification"
	"acilkan.backend/internal/repository"
	"acilkan.backend/internal/router"
	"acilkan.backend/internal/service"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/logger"
	"acilkan.backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	if err := logger.InitLogger(cfg.Env); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Emergency Blood Donation API",
		zap.String("version", router.APIVersion),
		zap.String("env", cfg.Env),
		zap.String("port", cfg.Port),
	)

	// Initialize Firebase Messaging client
	ctx := context.Background()
	messagingClient, err := cfg.FirebaseApp.Messaging(ctx)
	if err != nil {
		logger.Fatal("Failed to initialize Firebase Messaging", zap.Error(err))
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(cfg.FirestoreClient)
	bloodRequestRepo := repository.NewBloodRequestRepository(cfg.FirestoreClient)

	// Initialize notification client
	fcmClient := notification.NewFCMClient(messagingClient)

	// Initialize services
	userService := service.NewUserService(userRepo, fcmClient)
	donationService := service.NewDonationService(bloodRequestRepo, userRepo, fcmClient)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	requestHandler := handler.NewRequestHandler(donationService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Emergency Blood Donation API",
		ErrorHandler: customErrorHandler,
		BodyLimit:    64 * 1024,
	})

	// Global middleware
	app.Use(recover.New())
	// Per-IP rate limit as a basic abuse guard
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.RateLimitPerMinute,
		Expiration: time.Minute,
		Next: func(c *fiber.Ctx) bool {
			return c.Path() == "/health"
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.TooManyRequests(c, appErrors.ErrCodeRequestLimitExceeded, "Too many requests, please slow down")
		},
	}))
	// Mobile clients don't send Origin, so CORS only matters for browsers.
	// Restrict via CORS_ALLOW_ORIGINS in production (comma separated).
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
	
	// Reject CONNECT requests (from IDE proxies) silently
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == "CONNECT" {
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}
		return c.Next()
	})
	
	app.Use(logger.HTTPMiddleware())

	// Setup all routes
	router.SetupRoutes(app, cfg, userHandler, requestHandler)

	// Start server in a goroutine
	go func() {
		addr := ":" + cfg.Port
		logger.Info("Server starting", zap.String("address", addr))
		if err := app.Listen(addr); err != nil {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// Close Firestore client
	if err := cfg.FirestoreClient.Close(); err != nil {
		logger.Error("Failed to close Firestore client", zap.Error(err))
	}

	logger.Info("Server exited")
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Check if it's an AppError
	if appErrors.IsAppError(err) {
		appErr := appErrors.GetAppError(err)
		logger.Error("Application error",
			zap.String("path", c.Path()),
			zap.String("method", c.Method()),
			zap.String("error_code", appErr.Code),
			zap.String("error_message", appErr.Message),
			zap.Error(err),
		)
		return response.Error(c, code, &response.ErrorInfo{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
	}

	// Generic error
	logger.Error("Request error",
		zap.String("path", c.Path()),
		zap.String("method", c.Method()),
		zap.Int("status", code),
		zap.Error(err),
	)

	// Don't leak internal error details to clients; Fiber errors (404, 405...) are safe to show
	message := appErrors.ErrorMessages[appErrors.ErrCodeInternalError]
	if e, ok := err.(*fiber.Error); ok && code < fiber.StatusInternalServerError {
		message = e.Message
	}
	return response.Error(c, code, &response.ErrorInfo{
		Code:    appErrors.ErrCodeInternalError,
		Message: message,
	})
}
