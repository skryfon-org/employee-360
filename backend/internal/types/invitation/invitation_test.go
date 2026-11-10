package invitation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

func TestToInvitationResponse_Status(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	tests := []struct {
		name string
		inv  entity.UserInvitation
		want entity.InvitationStatus
	}{
		{"pending", entity.UserInvitation{ExpiresAt: future}, entity.InvitationStatusPending},
		{"accepted", entity.UserInvitation{ExpiresAt: future, AcceptedAt: &past}, entity.InvitationStatusAccepted},
		{"revoked", entity.UserInvitation{ExpiresAt: future, RevokedAt: &past}, entity.InvitationStatusRevoked},
		{"expired", entity.UserInvitation{ExpiresAt: past}, entity.InvitationStatusExpired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToInvitationResponse(&tt.inv).Status; got != tt.want {
				t.Fatalf("got %s want %s", got, tt.want)
			}
		})
	}
}

func TestToInvitationResponse_MapsFieldsAndHidesSecrets(t *testing.T) {
	dept := uuid.New()
	inv := &entity.UserInvitation{
		ID: uuid.New(), TenantID: uuid.New(), Email: "a@b.com", RoleID: uuid.New(),
		DepartmentID: &dept, InvitedBy: uuid.New(), TokenHash: "secret-hash",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	resp := ToInvitationResponse(inv)
	if resp.ID != inv.ID || resp.Email != inv.Email || resp.RoleID != inv.RoleID ||
		resp.InvitedBy != inv.InvitedBy || resp.DepartmentID == nil || *resp.DepartmentID != dept {
		t.Errorf("fields not mapped: %+v", resp)
	}
	raw, _ := json.Marshal(resp)
	for _, leak := range []string{"secret-hash", "token", "tenant_id"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("json must not contain %q: %s", leak, raw)
		}
	}
}
