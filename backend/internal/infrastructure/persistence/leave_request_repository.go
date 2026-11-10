package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	"github.com/skryfon-org/employee-360/backend/internal/infrastructure/database"
)

type gormLeaveRequestRepository struct {
	db *gorm.DB
}

// NewGormLeaveRequestRepository constructs a GORM-backed LeaveRequestRepository.
func NewGormLeaveRequestRepository(db *gorm.DB) repository.LeaveRequestRepository {
	return &gormLeaveRequestRepository{db: db}
}

var _ repository.LeaveRequestRepository = (*gormLeaveRequestRepository)(nil)

func (r *gormLeaveRequestRepository) scoped(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
}

// detailQuery joins requests (alias lr) with applicant, approver and leave type names.
// Joins are tenant-scoped; soft-deleted users/types still resolve for history.
func (r *gormLeaveRequestRepository) detailQuery(c context.Context, tenantID uuid.UUID) *gorm.DB {
	return database.DBFromContext(c, r.db).
		Table("leave_requests AS lr").
		Select(`lr.*,
			TRIM(u.first_name || ' ' || u.last_name) AS user_name,
			u.email AS user_email,
			TRIM(a.first_name || ' ' || a.last_name) AS approver_name,
			COALESCE(TRIM(d.first_name || ' ' || d.last_name), '') AS decided_by_name,
			lt.name AS leave_type_name,
			lt.code AS leave_type_code`).
		Joins("JOIN users u ON u.id = lr.user_id AND u.tenant_id = lr.tenant_id").
		Joins("JOIN users a ON a.id = lr.approver_id AND a.tenant_id = lr.tenant_id").
		Joins("LEFT JOIN users d ON d.id = lr.decided_by AND d.tenant_id = lr.tenant_id").
		Joins("JOIN leave_types lt ON lt.id = lr.leave_type_id AND lt.tenant_id = lr.tenant_id").
		Where("lr.tenant_id = ? AND lr.deleted_at IS NULL", tenantID)
}

func applyLeaveFilter(q *gorm.DB, f repository.LeaveRequestFilter) *gorm.DB {
	if f.Status != "" {
		q = q.Where("lr.status = ?", f.Status)
	}
	if f.ExcludeStatus != "" {
		q = q.Where("lr.status <> ?", f.ExcludeStatus)
	}
	if f.UserID != nil {
		q = q.Where("lr.user_id = ?", *f.UserID)
	}
	if f.ApproverID != nil {
		q = q.Where("lr.approver_id = ?", *f.ApproverID)
	}
	if f.LeaveTypeID != nil {
		q = q.Where("lr.leave_type_id = ?", *f.LeaveTypeID)
	}
	if f.From != nil {
		q = q.Where("lr.end_date >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("lr.start_date <= ?", *f.To)
	}
	return q
}

func (r *gormLeaveRequestRepository) LockUser(c context.Context, tenantID, userID uuid.UUID) error {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return domainerrors.ErrUnauthorized
	}
	key := fmt.Sprintf("leave_request:%s:%s", tenantID, userID)
	return database.DBFromContext(c, r.db).Exec("SELECT pg_advisory_xact_lock(hashtext(?))", key).Error
}

func (r *gormLeaveRequestRepository) Create(c context.Context, tenantID, actorID uuid.UUID, lr *entity.LeaveRequest) error {
	if tenantID == uuid.Nil {
		return domainerrors.ErrLeaveRequestNotFound
	}
	now := time.Now().UTC()
	if lr.ID == uuid.Nil {
		lr.ID = uuid.New()
	}
	lr.TenantID = tenantID
	lr.CreatedAt, lr.UpdatedAt = now, now
	lr.CreatedBy, lr.UpdatedBy = &actorID, &actorID
	return database.DBFromContext(c, r.db).Create(lr).Error
}

func (r *gormLeaveRequestRepository) get(c context.Context, tenantID, id uuid.UUID, lock bool) (*entity.LeaveRequest, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrLeaveRequestNotFound
	}
	q := r.scoped(c, tenantID)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var lr entity.LeaveRequest
	if err := q.Where("id = ?", id).First(&lr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrLeaveRequestNotFound
		}
		return nil, err
	}
	return &lr, nil
}

func (r *gormLeaveRequestRepository) GetByID(c context.Context, tenantID, id uuid.UUID) (*entity.LeaveRequest, error) {
	return r.get(c, tenantID, id, false)
}

func (r *gormLeaveRequestRepository) GetByIDForUpdate(c context.Context, tenantID, id uuid.UUID) (*entity.LeaveRequest, error) {
	return r.get(c, tenantID, id, true)
}

func (r *gormLeaveRequestRepository) GetDetail(c context.Context, tenantID, id uuid.UUID) (*entity.LeaveRequestDetail, error) {
	if tenantID == uuid.Nil || id == uuid.Nil {
		return nil, domainerrors.ErrLeaveRequestNotFound
	}
	var d entity.LeaveRequestDetail
	if err := r.detailQuery(c, tenantID).Where("lr.id = ?", id).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrLeaveRequestNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *gormLeaveRequestRepository) UpdateStatus(c context.Context, tenantID, actorID uuid.UUID, lr *entity.LeaveRequest) error {
	return r.update(c, tenantID, actorID, lr, false)
}

func (r *gormLeaveRequestRepository) RecordDecision(c context.Context, tenantID, actorID uuid.UUID, lr *entity.LeaveRequest) error {
	return r.update(c, tenantID, actorID, lr, true)
}

func (r *gormLeaveRequestRepository) update(c context.Context, tenantID, actorID uuid.UUID, lr *entity.LeaveRequest, withDecision bool) error {
	if tenantID == uuid.Nil || lr.ID == uuid.Nil {
		return domainerrors.ErrLeaveRequestNotFound
	}
	lr.UpdatedAt = time.Now().UTC()
	lr.UpdatedBy = &actorID
	cols := map[string]any{
		"status":     lr.Status,
		"updated_at": lr.UpdatedAt,
		"updated_by": actorID,
	}
	if withDecision {
		cols["decided_by"] = lr.DecidedBy
		cols["decided_at"] = lr.DecidedAt
		cols["decision_comment"] = lr.DecisionComment
	}
	res := r.scoped(c, tenantID).Model(&entity.LeaveRequest{}).Where("id = ?", lr.ID).Updates(cols)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domainerrors.ErrLeaveRequestNotFound
	}
	return nil
}

func (r *gormLeaveRequestRepository) HasOverlap(c context.Context, tenantID, userID uuid.UUID, start, end time.Time, dayPart string, excludeID uuid.UUID) (bool, error) {
	if tenantID == uuid.Nil || userID == uuid.Nil {
		return false, nil
	}
	q := r.scoped(c, tenantID).Model(&entity.LeaveRequest{}).
		Where("user_id = ? AND status IN ?", userID, []string{entity.LeaveStatusPending, entity.LeaveStatusApproved}).
		Where("start_date <= ? AND end_date >= ?", end, start)
	if start.Equal(end) && dayPart != entity.LeaveDayPartFull {
		// Opposite halves of the same single day coexist.
		q = q.Where("NOT (start_date = end_date AND day_part <> 'full' AND day_part <> ?)", dayPart)
	}
	if excludeID != uuid.Nil {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *gormLeaveRequestRepository) ListByUser(c context.Context, tenantID, userID uuid.UUID, f repository.LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error) {
	if userID == uuid.Nil {
		return []*entity.LeaveRequestDetail{}, 0, nil
	}
	f.UserID = &userID
	return r.List(c, tenantID, f, limit, offset)
}

func (r *gormLeaveRequestRepository) ListByApprover(c context.Context, tenantID, approverID uuid.UUID, f repository.LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error) {
	if approverID == uuid.Nil {
		return []*entity.LeaveRequestDetail{}, 0, nil
	}
	f.ApproverID = &approverID
	return r.List(c, tenantID, f, limit, offset)
}

// ApproverSummary counts pending and total requests addressed to approverID in
// one aggregate query, scoped like ListByApprover.
func (r *gormLeaveRequestRepository) ApproverSummary(c context.Context, tenantID, approverID uuid.UUID) (int64, bool, error) {
	if tenantID == uuid.Nil || approverID == uuid.Nil {
		return 0, false, nil
	}
	var row struct {
		Pending int64
		Total   int64
	}
	err := database.DBFromContext(c, r.db).Table("leave_requests AS lr").
		Select("COUNT(*) FILTER (WHERE lr.status = ?) AS pending, COUNT(*) AS total", entity.LeaveStatusPending).
		Where("lr.tenant_id = ? AND lr.deleted_at IS NULL AND lr.approver_id = ?", tenantID, approverID).
		Scan(&row).Error
	if err != nil {
		return 0, false, err
	}
	return row.Pending, row.Total > 0, nil
}

func (r *gormLeaveRequestRepository) List(c context.Context, tenantID uuid.UUID, f repository.LeaveRequestFilter, limit, offset int) ([]*entity.LeaveRequestDetail, int64, error) {
	if tenantID == uuid.Nil {
		return []*entity.LeaveRequestDetail{}, 0, nil
	}
	var total int64
	cq := applyLeaveFilter(database.DBFromContext(c, r.db).Table("leave_requests AS lr").
		Where("lr.tenant_id = ? AND lr.deleted_at IS NULL", tenantID), f)
	if err := cq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := applyLeaveFilter(r.detailQuery(c, tenantID), f)
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	out := []*entity.LeaveRequestDetail{}
	order := "lr.start_date DESC, lr.created_at DESC, lr.id ASC"
	if f.OldestFirst {
		order = "lr.start_date ASC, lr.created_at ASC, lr.id ASC"
	}
	if err := q.Order(order).Find(&out).Error; err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
