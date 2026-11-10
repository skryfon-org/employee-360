package project

import (
	"context"

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

type updateProjectUseCase struct {
	repo       repository.ProjectRepository
	audit      service.AuditRecorder
	transactor ucshared.Transactor
}

// NewUpdateProjectUseCase creates an UpdateProjectUseCase.
func NewUpdateProjectUseCase(repo repository.ProjectRepository, audit service.AuditRecorder, transactor ucshared.Transactor) prjuc.UpdateProjectUseCase {
	if transactor == nil {
		transactor = ucshared.NewNopTransactor()
	}
	return &updateProjectUseCase{repo: repo, audit: audit, transactor: transactor}
}

func (uc *updateProjectUseCase) Execute(c context.Context, tenantID, actorID uuid.UUID, in prjtypes.UpdateProjectInput) (*entity.Project, error) {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return nil, domainerrors.ErrUnauthorized
	}
	if in.ID == uuid.Nil {
		return nil, domainerrors.ErrProjectNotFound
	}
	var out *entity.Project
	err := uc.transactor.WithinTransaction(c, func(txCtx context.Context) error {
		p, err := uc.repo.GetByIDForUpdate(txCtx, tenantID, in.ID)
		if err != nil {
			return err
		}
		oldStatus, oldCode := p.Status, p.Code

		if in.Name.Set {
			if in.Name.Null {
				return ErrNameRequired
			}
			if p.Name, err = normalizeName(in.Name.Value); err != nil {
				return err
			}
		}
		if in.Code.Set {
			if in.Code.Null {
				return ErrCodeRequired
			}
			if p.Code, err = normalizeCode(in.Code.Value); err != nil {
				return err
			}
		}
		if in.Description.Set {
			if in.Description.Null {
				p.Description = ""
			} else if p.Description, err = normalizeDescription(in.Description.Value); err != nil {
				return err
			}
		}
		if in.ClientName.Set {
			if in.ClientName.Null {
				p.ClientName = nil
			} else {
				v := in.ClientName.Value
				if p.ClientName, err = projectcommon.NormalizeOptional(&v, 150, ErrClientNameTooLong); err != nil {
					return err
				}
			}
		}
		if in.Status.Set {
			if in.Status.Null || !validStatus(in.Status.Value) {
				return ErrStatusInvalid
			}
			p.Status = in.Status.Value
		}
		if in.StartDate.Set {
			if in.StartDate.Null || in.StartDate.Value.IsZero() {
				return projectcommon.ErrStartDateRequired
			}
			p.StartDate = projectcommon.DateOnly(in.StartDate.Value.Time)
		}
		if in.EndDate.Set {
			if in.EndDate.Null {
				p.EndDate = nil
			} else {
				e := projectcommon.DateOnly(in.EndDate.Value.Time)
				p.EndDate = &e
			}
		}
		if p.EndDate != nil && p.EndDate.Before(p.StartDate) {
			return ErrProjectDateOrder
		}

		if p.Code != oldCode {
			taken, err := uc.repo.ExistsByCode(txCtx, tenantID, p.Code)
			if err != nil {
				return err
			}
			if taken {
				return domainerrors.ErrProjectCodeTaken
			}
		}
		if err := uc.repo.Update(txCtx, tenantID, actorID, p); err != nil {
			return err
		}
		if err := uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionProjectUpdate, domainaudit.EntityProject, p.ID, map[string]any{
			"name": p.Name, "code": p.Code, "status": p.Status,
		}); err != nil {
			return err
		}
		if p.Status != oldStatus {
			if err := uc.audit.Record(txCtx, tenantID, actorID, domainaudit.ActionProjectStatusChange, domainaudit.EntityProject, p.ID, map[string]any{
				"from": oldStatus, "to": p.Status,
			}); err != nil {
				return err
			}
		}
		out = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
