package entity

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a tracked refresh token for session management and rotation.
type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	Family    uuid.UUID  `json:"family"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	IPAddress string     `json:"ip_address,omitempty"`
	UserAgent string     `json:"user_agent,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// IsActive returns true if the refresh token is not revoked and not expired.
func (rt *RefreshToken) IsActive() bool {
	return rt.RevokedAt == nil && time.Now().Before(rt.ExpiresAt)
}
