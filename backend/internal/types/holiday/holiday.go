// Package holiday holds API DTOs and usecase input types for holiday operations.
package holiday

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// CreateHolidayRequest contains fields for creating a holiday.
// Tenant identity is strictly derived from the request context.
type CreateHolidayRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required"`
	Name        string    `json:"name" binding:"required,min=1,max=150"`
	Description string    `json:"description" binding:"max=500"`
	Date        string    `json:"date" binding:"required"` // Format: YYYY-MM-DD
}

// UpdateHolidayRequest contains fields for updating a holiday.
// Tenant identity is strictly derived from the request context.
type UpdateHolidayRequest struct {
	CategoryID  uuid.UUID `json:"category_id" binding:"required"`
	Name        string    `json:"name" binding:"required,min=1,max=150"`
	Description string    `json:"description" binding:"max=500"`
	Date        string    `json:"date" binding:"required"` // Format: YYYY-MM-DD
}

// HolidayResponse represents the holiday API presentation model.
type HolidayResponse struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	CategoryID uuid.UUID `json:"category_id"`
	// CategoryIsPublic is true when the holiday's category is a public-holiday (company day off) category.
	CategoryIsPublic bool      `json:"category_is_public"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Date             string    `json:"date"` // Format: YYYY-MM-DD
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ToHolidayResponse maps a domain Holiday entity to HolidayResponse.
func ToHolidayResponse(h *entity.Holiday) HolidayResponse {
	return HolidayResponse{
		ID:               h.ID,
		TenantID:         h.TenantID,
		CategoryID:       h.CategoryID,
		CategoryIsPublic: h.CategoryIsPublic,
		Name:             h.Name,
		Description:      h.Description,
		Date:             h.Date.Format("2006-01-02"),
		CreatedAt:        h.CreatedAt,
		UpdatedAt:        h.UpdatedAt,
	}
}

// CreateHolidayInput holds validated data required to create a holiday.
type CreateHolidayInput struct {
	CategoryID  uuid.UUID
	Name        string
	Description string
	Date        time.Time
}

// UpdateHolidayInput specifies the holiday to modify and its new attributes.
type UpdateHolidayInput struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Name        string
	Description string
	Date        time.Time
}

// DeleteHolidayInput specifies the holiday to remove.
type DeleteHolidayInput struct {
	ID uuid.UUID
}

// GetHolidayQuery specifies the holiday to look up.
type GetHolidayQuery struct {
	ID uuid.UUID
}

// ListHolidaysByYearQuery holds parameters for listing holidays for a given year.
type ListHolidaysByYearQuery struct {
	Year       int
	CategoryID *uuid.UUID
	Month      *int
}
