package handler

import (
	"acilkan.backend/internal/auth"
	"acilkan.backend/internal/service"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile returns the authenticated user's profile
// GET /api/v1/users/me
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	uid := auth.GetUserUID(c)
	email := auth.GetUserEmail(c)

	user, err := h.userService.GetOrCreateUser(c.Context(), uid, email)
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeInternalError)
	}

	return response.Success(c, user)
}

// UpdateProfile updates the authenticated user's profile
// PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	uid := auth.GetUserUID(c)

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.BadRequest(c, appErrors.ErrCodeInvalidRequestBody, "Invalid request body")
	}

	user, err := h.userService.UpdateProfile(c.Context(), uid, updates)
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeUserUpdateFailed)
	}

	return response.Success(c, user)
}

// UpdateFCMToken updates the user's FCM token for push notifications
// POST /api/v1/users/fcm-token
func (h *UserHandler) UpdateFCMToken(c *fiber.Ctx) error {
	uid := auth.GetUserUID(c)

	var req struct {
		Token string `json:"token"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, appErrors.ErrCodeInvalidRequestBody, "Invalid request body")
	}

	if req.Token == "" {
		return response.BadRequest(c, appErrors.ErrCodeInvalidFCMToken, "FCM token is required")
	}

	if err := h.userService.UpdateFCMToken(c.Context(), uid, req.Token); err != nil {
		return writeError(c, err, appErrors.ErrCodeInternalError)
	}

	return response.Success(c, fiber.Map{
		"message": "FCM token updated successfully",
	})
}
