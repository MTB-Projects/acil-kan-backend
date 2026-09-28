package handler

import (
	"acilkan.backend/internal/hospital"
	"acilkan.backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

const (
	defaultHospitalLimit = 20
	maxHospitalLimit     = 50
)

type HospitalHandler struct {
	directory *hospital.Directory
}

func NewHospitalHandler(directory *hospital.Directory) *HospitalHandler {
	return &HospitalHandler{directory: directory}
}

// SearchHospitals returns hospitals for autocomplete.
// GET /api/v1/public/hospitals?city=Ankara&district=Çankaya&q=sehir&limit=20
func (h *HospitalHandler) SearchHospitals(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", defaultHospitalLimit)
	if limit <= 0 || limit > maxHospitalLimit {
		limit = defaultHospitalLimit
	}
	results := h.directory.Search(c.Query("city"), c.Query("district"), c.Query("q"), limit)
	return response.Success(c, results)
}
