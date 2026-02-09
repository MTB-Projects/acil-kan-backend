package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Response represents a standard API response
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta contains additional metadata for the response
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// Success sends a successful response
func Success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// SuccessWithMeta sends a successful response with metadata
func SuccessWithMeta(c *fiber.Ctx, data interface{}, meta *Meta) error {
	return c.JSON(Response{
		Success:   true,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Created sends a 201 Created response
func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Error sends an error response
func Error(c *fiber.Ctx, statusCode int, err *ErrorInfo) error {
	return c.Status(statusCode).JSON(Response{
		Success:   false,
		Error:     err,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *fiber.Ctx, code, message string) error {
	return Error(c, fiber.StatusBadRequest, &ErrorInfo{
		Code:    code,
		Message: message,
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *fiber.Ctx, code, message string) error {
	return Error(c, fiber.StatusUnauthorized, &ErrorInfo{
		Code:    code,
		Message: message,
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *fiber.Ctx, code, message string) error {
	return Error(c, fiber.StatusForbidden, &ErrorInfo{
		Code:    code,
		Message: message,
	})
}

// NotFound sends a 404 Not Found response
func NotFound(c *fiber.Ctx, code, message string) error {
	return Error(c, fiber.StatusNotFound, &ErrorInfo{
		Code:    code,
		Message: message,
	})
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(c *fiber.Ctx, code, message string) error {
	return Error(c, fiber.StatusInternalServerError, &ErrorInfo{
		Code:    code,
		Message: message,
	})
}

// TooManyRequests sends a 429 Too Many Requests response
func TooManyRequests(c *fiber.Ctx, code, message string) error {
	return Error(c, fiber.StatusTooManyRequests, &ErrorInfo{
		Code:    code,
		Message: message,
	})
}
