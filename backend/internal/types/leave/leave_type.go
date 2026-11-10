// Package leave holds API DTOs and usecase inputs/results for leave types and
// leave requests. Tenant identity is never part of a payload.
package leave

import (
	"time"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// CreateLeaveTypeRequest is the body of POST /leave-types.
type CreateLeaveTypeRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	// IsActive is optional and defaults to true.
	IsActive *bool `json:"is_active"`
}

// UpdateLeaveTypeRequest is the body of PATCH /leave-types/:id. Absent fields
// are unchanged. Leave types are deactivated (is_active=false), never deleted.
type UpdateLeaveTypeRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

// CreateLeaveTypeInput is the usecase input for creating a leave type.
type CreateLeaveTypeInput struct {
	Name, Code, Description string
	// IsActive nil means active (default true).
	IsActive *bool
}

// UpdateLeaveTypeInput is the usecase input for patching a leave type.
type UpdateLeaveTypeInput struct {
	Name, Code, Description *string
	IsActive                *bool
}

// LeaveTypeResponse is a leave type on the wire.
type LeaveTypeResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToLeaveTypeResponse maps an entity to its response.
func ToLeaveTypeResponse(lt *entity.LeaveType) LeaveTypeResponse {
	return LeaveTypeResponse{
		ID: lt.ID, Name: lt.Name, Code: lt.Code, Description: lt.Description,
		IsActive: lt.IsActive, CreatedAt: lt.CreatedAt, UpdatedAt: lt.UpdatedAt,
	}
}

// ToLeaveTypeResponses maps a slice of entities.
func ToLeaveTypeResponses(in []*entity.LeaveType) []LeaveTypeResponse {
	out := make([]LeaveTypeResponse, 0, len(in))
	for _, lt := range in {
		out = append(out, ToLeaveTypeResponse(lt))
	}
	return out
}
