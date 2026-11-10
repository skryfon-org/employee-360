package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

// gormUserInvitationRepository is a GORM-backed adapter implementing
// repository.UserInvitationRepository. Like the sibling repositories, every
// tenant-scoped method takes an explicit tenantID supplied by the usecase.
type gormUserInvitationRepository struct {
	db *gorm.DB
}

// NewGormUserInvitationRepository constructs a GORM-backed UserInvitationRepository.
func NewGormUserInvitationRepository(db *gorm.DB) repository.UserInvitationRepository {
	return &gormUserInvitationRepository{db: db}
}

var _ repository.UserInvitationRepository = (*gormUserInvitationRepository)(nil)

// pendingWhere restricts an update to a still-pending invitation in a tenant.
const pendingWhere = "id = ? AND tenant_id = ? AND deleted_at IS NULL AND accepted_at IS NULL AND revoked_at IS NULL"

func (r *gormUserInvitationRepository) Create(c context.Context, inv *entity.UserInvitation) error {
	return database.DBFromContext(c, r.db).Create(inv).Error
}

func (r *gormUserInvitationRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.UserInvitation, error) {
	var inv entity.UserInvitation
	if err := database.DBFromContext(c, r.db).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrInvitationNotFound
		}
		return nil, err
	}
	return &inv, nil
}

// GetByTokenHash is the only unscoped lookup: the accept flow is
// unauthenticated and resolves the tenant from the token itself.
func (r *gormUserInvitationRepository) GetByTokenHash(c context.Context, tokenHash string) (*entity.UserInvitation, error) {
	var inv entity.UserInvitation
	if err := database.DBFromContext(c, r.db).
		Where("token_hash = ? AND deleted_at IS NULL", tokenHash).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrInvitationNotFound
		}
		return nil, err
	}
	return &inv, nil
}

// conditionalUpdate applies updates only while the invitation is pending and
// returns ErrInvitationNotPending when no row matched (missing, foreign
// tenant, accepted or revoked).
func (r *gormUserInvitationRepository) conditionalUpdate(c context.Context, tenantID, id uuid.UUID, updates map[string]any) error {
	res := database.DBFromContext(c, r.db).
		Model(&entity.UserInvitation{}).
		Where(pendingWhere, id, tenantID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrInvitationNotPending
	}
	return nil
}

func (r *gormUserInvitationRepository) UpdateToken(c context.Context, tenantID, id, actorID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	return r.conditionalUpdate(c, tenantID, id, map[string]any{
		"token_hash": tokenHash,
		"expires_at": expiresAt,
		"updated_at": time.Now().UTC(),
		"updated_by": actorID,
	})
}

func (r *gormUserInvitationRepository) MarkAccepted(c context.Context, tenantID, id uuid.UUID, at time.Time) error {
	return r.conditionalUpdate(c, tenantID, id, map[string]any{"accepted_at": at, "updated_at": at})
}

func (r *gormUserInvitationRepository) MarkRevoked(c context.Context, tenantID, id, actorID uuid.UUID, at time.Time) error {
	return r.conditionalUpdate(c, tenantID, id, map[string]any{"revoked_at": at, "updated_at": at, "updated_by": actorID})
}

func (r *gormUserInvitationRepository) HasAcceptedForEmail(c context.Context, tenantID uuid.UUID, email string) (bool, error) {
	var n int64
	if err := database.DBFromContext(c, r.db).
		Model(&entity.UserInvitation{}).
		Where("tenant_id = ? AND email = ? AND deleted_at IS NULL AND accepted_at IS NOT NULL", tenantID, email).
		Limit(1).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *gormUserInvitationRepository) RevokeUnacceptedForEmail(c context.Context, tenantID uuid.UUID, email string, actorID uuid.UUID, at time.Time) (int64, error) {
	res := database.DBFromContext(c, r.db).
		Model(&entity.UserInvitation{}).
		Where("tenant_id = ? AND email = ? AND deleted_at IS NULL AND accepted_at IS NULL AND revoked_at IS NULL", tenantID, email).
		Updates(map[string]any{"revoked_at": at, "updated_at": at, "updated_by": actorID})
	return res.RowsAffected, res.Error
}

// likeEscaper escapes LIKE wildcards so user input is matched literally.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type invitationListRow struct {
	entity.UserInvitation
	RoleName       string
	InvitedByFirst string
	InvitedByLast  string
	InvitedByEmail string
}

func (r *gormUserInvitationRepository) List(c context.Context, tenantID uuid.UUID, f repository.InvitationListFilter) ([]*entity.InvitationListItem, int64, error) {
	now := f.Now
	if now.IsZero() {
		now = time.Now()
	}
	q := database.DBFromContext(c, r.db).Table("user_invitations AS i").
		Where("i.tenant_id = ? AND i.deleted_at IS NULL", tenantID)
	if s := strings.TrimSpace(f.Search); s != "" {
		q = q.Where(`i.email ILIKE ? ESCAPE '\'`, "%"+likeEscaper.Replace(s)+"%")
	}
	switch f.Status {
	case entity.InvitationStatusAccepted:
		q = q.Where("i.accepted_at IS NOT NULL")
	case entity.InvitationStatusRevoked:
		q = q.Where("i.accepted_at IS NULL AND i.revoked_at IS NOT NULL")
	case entity.InvitationStatusExpired:
		q = q.Where("i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at <= ?", now)
	case entity.InvitationStatusPending:
		q = q.Where("i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > ?", now)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []invitationListRow
	pq := q.Session(&gorm.Session{}).
		Select(`i.*, r.name AS role_name, u.first_name AS invited_by_first,
			u.last_name AS invited_by_last, u.email AS invited_by_email`).
		// LEFT JOINs keep the invitation listed even when its role or inviter is
		// soft-deleted; the deleted row's name/email then comes back empty.
		Joins("LEFT JOIN roles r ON r.id = i.role_id AND r.tenant_id = i.tenant_id AND r.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = i.invited_by AND u.tenant_id = i.tenant_id AND u.deleted_at IS NULL").
		Order("i.created_at DESC, i.id DESC")
	if f.Limit > 0 {
		pq = pq.Limit(f.Limit)
	}
	if f.Offset > 0 {
		pq = pq.Offset(f.Offset)
	}
	if err := pq.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*entity.InvitationListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.InvitationListItem{
			UserInvitation: row.UserInvitation, RoleName: row.RoleName,
			InvitedByFirst: row.InvitedByFirst, InvitedByLast: row.InvitedByLast, InvitedByEmail: row.InvitedByEmail,
		})
	}
	return out, total, nil
}
