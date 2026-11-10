// Package position holds API DTOs for position operations.
package position

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// CreatePositionRequest contains fields for creating a position.
// Tenant identity is strictly derived from the request context.
type CreatePositionRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	// IsActive is optional and defaults to true when omitted.
	IsActive *bool `json:"is_active,omitempty"`
}

// UpdatePositionRequest contains fields for updating a position.
// Tenant identity is strictly derived from the request context.
type UpdatePositionRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	// IsActive is optional; when omitted the current value is kept.
	IsActive *bool `json:"is_active,omitempty"`
}

// PositionResponse represents the position API presentation model.
type PositionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToPositionResponse maps a domain Position entity to PositionResponse.
func ToPositionResponse(p *entity.Position) PositionResponse {
	return PositionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// CreatePositionInput holds validated data required to create a position.
type CreatePositionInput struct {
	Name        string
	Description string
	IsActive    *bool // nil defaults to true
}

// UpdatePositionInput specifies the position to modify and its new attributes.
type UpdatePositionInput struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsActive    *bool // nil keeps the current value
}

// DeletePositionInput specifies the position to remove.
type DeletePositionInput struct {
	ID uuid.UUID
}

// GetPositionQuery specifies the position to look up.
type GetPositionQuery struct {
	ID uuid.UUID
}

// ListPositionsQuery holds pagination parameters for listing positions.
type ListPositionsQuery struct {
	Page     int
	PageSize int
	IsActive *bool // nil returns all positions
}

// ListPositionsResult contains paginated positions and pagination metadata.
type ListPositionsResult struct {
	Positions []*entity.Position
	Total     int64
	Page      int
	PageSize  int
}
