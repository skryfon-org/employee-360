package entity

import (
	"testing"
	"time"
)

func TestRefreshToken_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		token RefreshToken
		want  bool
	}{
		{
			name: "not revoked and not expired",
			token: RefreshToken{
				RevokedAt: nil,
				ExpiresAt: now.Add(time.Hour),
			},
			want: true,
		},
		{
			name: "revoked",
			token: RefreshToken{
				RevokedAt: timePtr(now.Add(-time.Minute)),
				ExpiresAt: now.Add(time.Hour),
			},
			want: false,
		},
		{
			name: "expired",
			token: RefreshToken{
				RevokedAt: nil,
				ExpiresAt: now.Add(-time.Minute),
			},
			want: false,
		},
		{
			name: "revoked and expired",
			token: RefreshToken{
				RevokedAt: timePtr(now.Add(-time.Minute)),
				ExpiresAt: now.Add(-time.Minute),
			},
			want: false,
		},
		{
			name: "exact-expiry boundary is treated as expired",
			token: RefreshToken{
				RevokedAt: nil,
				ExpiresAt: now,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}
