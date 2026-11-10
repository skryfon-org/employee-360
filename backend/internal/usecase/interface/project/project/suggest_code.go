package project

import (
	"context"

	"github.com/google/uuid"
)

// SuggestProjectCodeUseCase returns the code that creating a project with the
// given name (and no explicit code) would currently be assigned in the tenant.
// It is advisory only: the final code is decided at create time.
type SuggestProjectCodeUseCase interface {
	Execute(ctx context.Context, tenantID uuid.UUID, name string) (string, error)
}
