package entity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordResetToken_IsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		token PasswordResetToken
		want  bool
	}{
		{
			name: "not used and not expired",
			token: PasswordResetToken{
				UsedAt:    nil,
				ExpiresAt: now.Add(time.Hour),
			},
			want: true,
		},
		{
			name: "already used",
			token: PasswordResetToken{
				UsedAt:    timePtr(now.Add(-time.Minute)),
				ExpiresAt: now.Add(time.Hour),
			},
			want: false,
		},
		{
			name: "expired",
			token: PasswordResetToken{
				UsedAt:    nil,
				ExpiresAt: now.Add(-time.Minute),
			},
			want: false,
		},
		{
			name: "used and expired",
			token: PasswordResetToken{
				UsedAt:    timePtr(now.Add(-time.Minute)),
				ExpiresAt: now.Add(-time.Minute),
			},
			want: false,
		},
		{
			name: "exact-expiry boundary is treated as expired",
			token: PasswordResetToken{
				UsedAt:    nil,
				ExpiresAt: now,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.token.ID = uuid.New()
			if got := tt.token.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
