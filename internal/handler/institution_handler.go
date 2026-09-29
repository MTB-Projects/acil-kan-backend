package handler

import (
	"acilkan.backend/internal/auth"
	"acilkan.backend/internal/model"
	"acilkan.backend/internal/service"
	appErrors "acilkan.backend/pkg/errors"
	"acilkan.backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type InstitutionHandler struct {
	service *service.InstitutionService
}

func NewInstitutionHandler(s *service.InstitutionService) *InstitutionHandler {
	return &InstitutionHandler{service: s}
}

// Apply submits a verification application
// POST /api/v1/institution/applications
func (h *InstitutionHandler) Apply(c *fiber.Ctx) error {
	var in service.ApplyInput
	if err := c.BodyParser(&in); err != nil {
		return response.BadRequest(c, appErrors.ErrCodeInvalidRequestBody, "Invalid request body")
	}
	app, err := h.service.Apply(c.Context(), auth.GetUserUID(c), &in)
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}
	return response.Created(c, app)
}

// MyApplication returns the caller's latest application (data is null if none)
// GET /api/v1/institution/applications/me
func (h *InstitutionHandler) MyApplication(c *fiber.Ctx) error {
	app, err := h.service.MyApplication(c.Context(), auth.GetUserUID(c))
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}
	return response.Success(c, app)
}

// adminApplication adds reviewer hints to an application
type adminApplication struct {
	*model.InstitutionApplication
	InstitutionalEmail bool `json:"institutional_email"`
}

func withHints(apps ...*model.InstitutionApplication) []adminApplication {
	out := make([]adminApplication, 0, len(apps))
	for _, a := range apps {
		out = append(out, adminApplication{a, a.HasInstitutionalEmail()})
	}
	return out
}

// List returns applications for review (admin)
// GET /api/v1/admin/institution-applications?status=PENDING
func (h *InstitutionHandler) List(c *fiber.Ctx) error {
	st := model.ApplicationStatus(c.Query("status", string(model.ApplicationPending)))
	apps, err := h.service.List(c.Context(), st)
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}
	return response.Success(c, withHints(apps...))
}

// Approve verifies the applicant's institution (admin)
// POST /api/v1/admin/institution-applications/:id/approve
func (h *InstitutionHandler) Approve(c *fiber.Ctx) error {
	app, err := h.service.Approve(c.Context(), auth.GetUserUID(c), c.Params("id"))
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}
	return response.Success(c, withHints(app)[0])
}

// Reject declines the application with a reason shown to the applicant (admin)
// POST /api/v1/admin/institution-applications/:id/reject  {"reason": "..."}
func (h *InstitutionHandler) Reject(c *fiber.Ctx) error {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, appErrors.ErrCodeInvalidRequestBody, "Invalid request body")
	}
	app, err := h.service.Reject(c.Context(), auth.GetUserUID(c), c.Params("id"), body.Reason)
	if err != nil {
		return writeError(c, err, appErrors.ErrCodeDatabaseError)
	}
	return response.Success(c, withHints(app)[0])
}
