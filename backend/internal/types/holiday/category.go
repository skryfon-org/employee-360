// Package holiday holds API DTOs and usecase input types for holiday operations.
package holiday

import (
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// CreateCategoryRequest contains fields for creating a holiday category.
// Note: HTTP DTOs in this file are intentional pre-work for G5 (HTTP handlers).
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	Color       string `json:"color" binding:"required"`
	IsActive    *bool  `json:"is_active"`
	IsPublic    *bool  `json:"is_public"` // nil: false on create, unchanged on update
}

// UpdateCategoryRequest contains fields for updating a holiday category.
// Note: HTTP DTOs in this file are intentional pre-work for G5 (HTTP handlers).
type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	Color       string `json:"color" binding:"required"`
	IsActive    *bool  `json:"is_active"`
	IsPublic    *bool  `json:"is_public"` // nil: false on create, unchanged on update
}

// CategoryResponse represents the holiday category API presentation model.
// Note: HTTP DTOs in this file are intentional pre-work for G5 (HTTP handlers).
type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	IsActive    bool      `json:"is_active"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToCategoryResponse maps a domain HolidayCategory entity to CategoryResponse.
func ToCategoryResponse(c *entity.HolidayCategory) CategoryResponse {
	return CategoryResponse{
		ID:          c.ID,
		TenantID:    c.TenantID,
		Name:        c.Name,
		Description: c.Description,
		Color:       c.Color,
		IsActive:    c.IsActive,
		IsPublic:    c.IsPublic,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// CreateCategoryInput holds validated data required to create a holiday category.
type CreateCategoryInput struct {
	Name        string
	Description string
	Color       string
	IsActive    *bool
	IsPublic    *bool
}

// UpdateCategoryInput specifies the category to modify and its new attributes.
type UpdateCategoryInput struct {
	ID          uuid.UUID
	Name        string
	Description string
	Color       string
	IsActive    *bool
	IsPublic    *bool
}

// DeleteCategoryInput specifies the category to remove.
type DeleteCategoryInput struct {
	ID uuid.UUID
}

// GetCategoryQuery specifies the category to look up.
type GetCategoryQuery struct {
	ID uuid.UUID
}

// ListCategoriesQuery holds parameters for listing holiday categories.
type ListCategoriesQuery struct {
	Page     int
	PageSize int
	IsActive *bool // nil returns all categories
}

// ListCategoriesResult contains a page of categories and pagination metadata.
type ListCategoriesResult struct {
	Categories []*entity.HolidayCategory
	Total      int64
	Page       int
	PageSize   int
}
