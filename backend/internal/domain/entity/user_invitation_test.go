package entity

import (
	"testing"
	"time"
)

func TestUserInvitation_Status(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	tests := []struct {
		name string
		inv  UserInvitation
		want InvitationStatus
	}{
		{"pending", UserInvitation{ExpiresAt: future}, InvitationStatusPending},
		{"expired", UserInvitation{ExpiresAt: past}, InvitationStatusExpired},
		{"accepted", UserInvitation{ExpiresAt: future, AcceptedAt: &past}, InvitationStatusAccepted},
		{"revoked", UserInvitation{ExpiresAt: future, RevokedAt: &past}, InvitationStatusRevoked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.inv.Status(now); got != tt.want {
				t.Fatalf("got %s want %s", got, tt.want)
			}
		})
	}
}
