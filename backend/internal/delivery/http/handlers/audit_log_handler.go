package handlers

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/delivery/http/response"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	altypes "github.com/skryfon-org/employee-360/backend/internal/types/auditlog"
	aluc "github.com/skryfon-org/employee-360/backend/internal/usecase/interface/auditlog"
)

// AuditLogResponse is the audit-log API presentation model.
type AuditLogResponse = altypes.AuditLogResponse

// AuditLogHandler serves the read-only audit-log viewer.
type AuditLogHandler struct {
	listUC aluc.ListAuditLogsUseCase
	getUC  aluc.GetAuditLogUseCase
}

// NewAuditLogHandler constructs an AuditLogHandler.
func NewAuditLogHandler(listUC aluc.ListAuditLogsUseCase, getUC aluc.GetAuditLogUseCase) *AuditLogHandler {
	return &AuditLogHandler{listUC: listUC, getUC: getUC}
}

func writeAuditLogError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrAuditLogNotFound):
		response.NotFound(c, "audit log not found")
	case errors.Is(err, domainerrors.ErrInvalidAuditLogFilter):
		response.BadRequest(c, err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized):
		response.Unauthorized(c, "unauthorized")
	case errors.Is(err, domainerrors.ErrForbidden):
		response.Forbidden(c, "insufficient permissions")
	default:
		response.Internal(c, "an unexpected error occurred")
	}
}

func timeQuery(c *gin.Context, key string) (*time.Time, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		response.BadRequest(c, key+" must be an RFC3339 timestamp")
		return nil, false
	}
	return &t, true
}

// List returns a paginated, newest-first list of the caller tenant's audit entries.
//
// @Summary      List audit logs
// @Description  Read-only view of administrative changes, newest first. `action` is an exact action (e.g. department.create) or, when it ends with a dot, a prefix (e.g. department.). `from`/`to` are inclusive RFC3339 timestamps (URL-encode a `+` offset as %2B).
// @Tags         audit-logs
// @Produce      json
// @Security     BearerAuth
// @Param        page           query     int     false  "Page number (default 1)"
// @Param        page_size      query     int     false  "Page size (default 20, max 100)"
// @Param        action         query     string  false  "Exact action, or prefix when ending with '.'"
// @Param        actor_user_id  query     string  false  "Filter by actor user UUID"
// @Param        entity_type    query     string  false  "Filter by target entity type (e.g. department)"
// @Param        from           query     string  false  "Created at or after (RFC3339)"
// @Param        to             query     string  false  "Created at or before (RFC3339)"
// @Success      200            {object}  response.Envelope{data=[]altypes.AuditLogResponse,meta=response.Meta}
// @Failure      400            {object}  response.Envelope
// @Failure      401            {object}  response.Envelope
// @Failure      403            {object}  response.Envelope
// @Failure      404            {object}  response.Envelope
// @Router       /api/v1/audit-logs [get]
func (h *AuditLogHandler) List(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}

	page, pageSize, ok := pageParams(c)
	if !ok {
		return
	}

	var actorID *uuid.UUID
	if raw := c.Query("actor_user_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "actor_user_id must be a UUID")
			return
		}
		actorID = &id
	}
	from, ok := timeQuery(c, "from")
	if !ok {
		return
	}
	to, ok := timeQuery(c, "to")
	if !ok {
		return
	}

	out, err := h.listUC.Execute(c.Request.Context(), tenantID, altypes.ListAuditLogsQuery{
		Page:        page,
		PageSize:    pageSize,
		Action:      c.Query("action"),
		ActorUserID: actorID,
		EntityType:  c.Query("entity_type"),
		From:        from,
		To:          to,
	})
	if err != nil {
		writeAuditLogError(c, err)
		return
	}

	items := make([]altypes.AuditLogResponse, 0, len(out.Entries))
	for _, e := range out.Entries {
		items = append(items, altypes.ToAuditLogResponse(e))
	}
	response.Paginated(c, items, pageMeta(out.Page, out.PageSize, out.Total))
}

// GetByID returns one audit entry within the caller's tenant.
//
// @Summary      Get audit log by ID
// @Tags         audit-logs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Audit log UUID"
// @Success      200  {object}  response.Envelope{data=altypes.AuditLogResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /api/v1/audit-logs/{id} [get]
func (h *AuditLogHandler) GetByID(c *gin.Context) {
	tenantID, ok := tenantOnly(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid audit log id")
		return
	}
	e, err := h.getUC.Execute(c.Request.Context(), tenantID, altypes.GetAuditLogQuery{ID: id})
	if err != nil {
		writeAuditLogError(c, err)
		return
	}
	response.Success(c, altypes.ToAuditLogResponse(e))
}
