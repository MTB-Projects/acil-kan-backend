package handler

import (
	"acilkan.backend/internal/auth"
	"acilkan.backend/internal/model"
	"acilkan.backend/internal/service"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type RequestHandler struct {
	donationService *service.DonationService
}

func NewRequestHandler(donationService *service.DonationService) *RequestHandler {
	return &RequestHandler{donationService: donationService}
}

// CreateRequest creates a new blood donation request
// POST /api/v1/requests
func (h *RequestHandler) CreateRequest(c *fiber.Ctx) error {
	uid := auth.GetUserUID(c)

	var input service.CreateRequestInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, appErrors.ErrCodeInvalidRequestBody, "Invalid request body")
	}

	request, err := h.donationService.CreateRequest(c.Context(), uid, &input)
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeRequestCreationFailed)
	}

	return response.Created(c, request)
}

// GetActiveRequests returns all active blood requests with optional filters.
// Public: personal data is stripped.
// GET /api/v1/public/requests?city=Istanbul&district=Kadikoy&blood_type=A+
func (h *RequestHandler) GetActiveRequests(c *fiber.Ctx) error {
	// Parse query parameters for filtering
	city := c.Query("city")
	district := c.Query("district")
	bloodType := c.Query("blood_type")

	var requests []*model.PublicBloodRequest
	var err error

	// Use filtered query if any filter is provided
	if city != "" || district != "" || bloodType != "" {
		requests, err = h.donationService.GetActiveRequestsWithFilters(c.Context(), city, district, bloodType)
	} else {
		requests, err = h.donationService.GetActiveRequests(c.Context())
	}

	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}

	return response.Success(c, requests)
}

// GetRequest returns a single active request with contact details
// GET /api/v1/requests/:id
func (h *RequestHandler) GetRequest(c *fiber.Ctx) error {
	request, err := h.donationService.GetRequestContact(c.Context(), c.Params("id"))
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}
	return response.Success(c, request)
}

// GetMyRequests returns all requests created by the authenticated user
// GET /api/v1/requests/my
func (h *RequestHandler) GetMyRequests(c *fiber.Ctx) error {
	uid := auth.GetUserUID(c)

	requests, err := h.donationService.GetUserRequests(c.Context(), uid)
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}

	return response.Success(c, requests)
}

// CancelRequest cancels a blood request
// DELETE /api/v1/requests/:id
func (h *RequestHandler) CancelRequest(c *fiber.Ctx) error {
	uid := auth.GetUserUID(c)
	requestID := c.Params("id")

	if requestID == "" {
		return response.BadRequest(c, appErrors.ErrCodeMissingRequiredField, "Request ID is required")
	}

	if err := h.donationService.CancelRequest(c.Context(), uid, requestID); err != nil {
		return writeError(c, err, appErrors.ErrCodeRequestUpdateFailed)
	}

	return response.Success(c, fiber.Map{
		"message": "Request cancelled successfully",
	})
}
