// Package role holds API DTOs for role lookups.
package role

import (
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

// RoleResponse is the API representation of an assignable role (id + name only;
// the tenant ID is never exposed).
type RoleResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ToRoleResponse maps a role entity to its API representation.
func ToRoleResponse(r *entity.Role) RoleResponse {
	return RoleResponse{ID: r.ID, Name: r.Name}
}
