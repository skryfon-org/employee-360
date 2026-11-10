package project

import (
	"context"
	"errors"
	"strings"
	"time"

	projectcommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/project"

	"github.com/google/uuid"

	domainaudit "github.com/skryfon-org/employee-360/backend/internal/domain/audit"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/domain/service"
	prjtypes "github.com/skryfon-org/employee-360/backend/internal/types/project"
	"github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/ucshared"
	prjuc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/project/project"
)

type createProjectUseCase struct {
	repo       repository.ProjectRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewCreateProjectUseCase creates a CreateProjectUseCase.
func NewCreateProjectUseCase(repo repository.ProjectRepository, audit service.AuditRecorder, transactor ucshared.Transactor) prjuc.CreateProjectUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &createProjectUseCase{repo: repo, audit: audit, transactor: transactor}
}

func (uc *createProjectUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, in prjtypes.CreateProjectInput) (*entity.Project, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	name, err := normalizeName(in.Name)
	if err != nil {
		return nil, err
	}
	generated := strings.TrimSpace(in.Code) == ""
	var code string
	if !generated {
		if code, err = normalizeCode(in.Code); err != nil {
			return nil, err
		}
	}
	desc, err := normalizeDescription(in.Description)
	if err != nil {
		return nil, err
	}
	client, err := projectcommon.NormalizeOptional(in.ClientName, 150, ErrClientNameTooLong)
	if err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = entity.ProjectStatusActive
	}
	if !validStatus(status) {
		return nil, ErrStatusInvalid
	}
	if in.StartDate.IsZero() {
		return nil, projectcommon.ErrStartDateRequired
	}
	start := projectcommon.DateOnly(in.StartDate)
	var end *time.Time
	if in.EndDate != nil {
		e := projectcommon.DateOnly(*in.EndDate)
		if e.Before(start) {
			return nil, ErrProjectDateOrder
		}
		end = &e
	}

	// A generated code can lose a race against a concurrent insert (the unique
	// violation aborts the transaction), so the whole transaction is retried
	// with the next suffix. An explicit code returns the 409 immediately.
	suffixN := 1
	for attempt := 0; attempt < maxCodeAttempts; attempt++ {
		p := &entity.Project{
			ID: uuid.New(), TenantID: tenantID, Name: name, Code: code, Description: desc,
			ClientName: client, Status: status, StartDate: start, EndDate: end,
		}
		err = uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
			if generated {
				var gerr error
				if p.Code, suffixN, gerr = firstFreeCode(txCtx, uc.repo, tenantID, name, suffixN); gerr != nil {
					return gerr
				}
			} else {
				taken, err := uc.repo.ExistsByCode(txCtx, tenantID, p.Code)
				if err != nil {
					return err
				}
				if taken {
					return domainerrors.ErrProjectCodeTaken
				}
			}
			if err := uc.repo.Create(txCtx, tenantID, actorID, p); err != nil {
				return err
			}
			return uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionProjectCreate, domainaudit.EntityProject, p.ID, map[string]any{
				"name": p.Name, "code": p.Code, "status": p.Status,
			})
		})
		if err == nil {
			return p, nil
		}
		if generated && errors.Is(err, domainerrors.ErrProjectCodeTaken) {
			suffixN++
			continue
		}
		return nil, err
	}
	return nil, domainerrors.ErrProjectCodeTaken
}
