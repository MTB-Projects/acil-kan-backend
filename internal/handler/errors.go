package handler

import (
	"strings"

	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/logger"
	"acilkan.backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// writeError maps an error to an HTTP response. Internal error details are
// logged but never sent to the client.
func writeError(c *fiber.Ctx, err error, fallbackCode string) error {
	if !appErrors.IsAppError(err) {
		logger.Error("Unhandled error", zap.String("path", c.Path()), zap.Error(err))
		return response.InternalServerError(c, fallbackCode, appErrors.ErrorMessages[fallbackCode])
	}

	appErr := appErrors.GetAppError(err)
	switch {
	case appErr.Code == appErrors.ErrCodeUserNotFound, appErr.Code == appErrors.ErrCodeRequestNotFound,
		appErr.Code == appErrors.ErrCodeApplicationNotFound:
		return response.NotFound(c, appErr.Code, appErr.Message)
	case appErr.Code == appErrors.ErrCodeUnauthorizedAction, appErr.Code == appErrors.ErrCodeInsufficientPermission:
		return response.Forbidden(c, appErr.Code, appErr.Message)
	case appErr.Code == appErrors.ErrCodeRequestLimitExceeded:
		return response.TooManyRequests(c, appErr.Code, appErr.Message)
	case strings.HasPrefix(appErr.Code, "DB_"), strings.HasPrefix(appErr.Code, "GENERAL_"),
		appErr.Code == appErrors.ErrCodeRequestCreationFailed, appErr.Code == appErrors.ErrCodeRequestUpdateFailed,
		appErr.Code == appErrors.ErrCodeUserUpdateFailed, appErr.Code == appErrors.ErrCodeUserCreationFailed:
		logger.Error("Internal error", zap.String("path", c.Path()), zap.String("code", appErr.Code), zap.Error(err))
		return response.InternalServerError(c, appErr.Code, appErr.Message)
	default:
		return response.BadRequest(c, appErr.Code, appErr.Message)
	}
}
