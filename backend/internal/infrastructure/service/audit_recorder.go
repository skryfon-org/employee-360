package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
)

var (
	// ErrAuditInvalidIdentity is returned for a nil tenant or actor ID.
	ErrAuditInvalidIdentity = errors.New("audit: tenant and actor ids are required")
	// ErrAuditInvalidAction is returned when the action is not "entity.verb" shaped.
	ErrAuditInvalidAction = errors.New("audit: action must follow entity.verb naming")
	// ErrAuditInvalidEntityType is returned for an empty entity type.
	ErrAuditInvalidEntityType = errors.New("audit: entity type is required")
)

// actionPattern: two or more dot-separated lowercase snake_case segments.
var actionPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// sensitiveTokens are whole key tokens that mark a key as sensitive.
var sensitiveTokens = map[string]struct{}{
	"password": {}, "passwd": {}, "token": {}, "otp": {}, "secret": {},
	"authorization": {}, "credential": {}, "credentials": {},
	"apikey": {}, "privatekey": {},
}

// sensitivePairs are adjacent token pairs (joined by "_") that mark a key as
// sensitive, e.g. api_key / apiKey / api-key.
var sensitivePairs = map[string]struct{}{"api_key": {}, "private_key": {}}

// auditRecorder implements domainservice.AuditRecorder over AuditRepository. The repository
// picks up the caller's transaction from ctx.
type auditRecorder struct {
	repo repository.AuditRepository
	now  func() time.Time
}

// NewAuditRecorder returns a repository-backed AuditRecorder.
func NewAuditRecorder(repo repository.AuditRepository) domainservice.AuditRecorder {
	return &auditRecorder{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// Record validates, redacts and persists one audit entry.
func (r *auditRecorder) Record(ctx context.Context, tenantID, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, meta map[string]any) error {
	if tenantID == uuid.Nil || actorID == uuid.Nil {
		return ErrAuditInvalidIdentity
	}
	if !actionPattern.MatchString(action) {
		return ErrAuditInvalidAction
	}
	if strings.TrimSpace(entityType) == "" {
		return ErrAuditInvalidEntityType
	}
	raw, err := json.Marshal(redact(meta))
	if err != nil {
		return err
	}
	now := r.now()
	return r.repo.Create(ctx, &entity.AuditLog{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ActorUserID: &actorID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Metadata:    string(raw),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

// isSensitive reports whether key names a secret. The key is split into
// lowercase tokens on _ - . whitespace and camelCase boundaries, so "otp_code"
// and "accessToken" match while "footprint" and "hotpath" do not.
func isSensitive(key string) bool {
	tokens := tokenizeKey(key)
	for i, t := range tokens {
		if _, ok := sensitiveTokens[t]; ok {
			return true
		}
		if i+1 < len(tokens) {
			if _, ok := sensitivePairs[t+"_"+tokens[i+1]]; ok {
				return true
			}
		}
	}
	return false
}

// tokenizeKey splits on separators and camelCase boundaries (including
// acronym boundaries such as "OTPCode" -> otp, code) into lowercase tokens.
func tokenizeKey(key string) []string {
	var tokens []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			tokens = append(tokens, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	runes := []rune(key)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == '.' || unicode.IsSpace(r):
			flush()
			continue
		case unicode.IsUpper(r) && len(cur) > 0:
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return tokens
}

// redact returns a copy of meta with sensitive keys removed, recursing into
// nested maps and slices. A nil meta stays nil (marshals to "null" as before).
func redact(meta map[string]any) map[string]any {
	if meta == nil {
		return nil
	}
	out := make(map[string]any, len(meta))
	for k, v := range meta {
		if isSensitive(k) {
			continue
		}
		out[k] = redactValue(v)
	}
	return out
}

// redactValue returns a redacted copy of v without mutating it. Maps with
// string keys become map[string]any and slices/arrays become []any (JSON
// output is identical); other values pass through unchanged.
func redactValue(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case map[string]any:
		return redact(t)
	case []any:
		cp := make([]any, len(t))
		for i := range t {
			cp[i] = redactValue(t[i])
		}
		return cp
	}
	return redactReflect(reflect.ValueOf(v), v)
}

func redactReflect(rv reflect.Value, orig any) any {
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return orig
		}
		switch rv.Elem().Kind() {
		case reflect.Map, reflect.Slice, reflect.Array, reflect.Pointer, reflect.Interface:
			return redactReflect(rv.Elem(), rv.Elem().Interface())
		}
		return orig
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String || rv.IsNil() {
			return orig
		}
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			if isSensitive(k) {
				continue
			}
			out[k] = redactValue(iter.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return orig
		}
		if rv.Type().Elem().Kind() == reflect.Uint8 { // []byte: opaque
			return orig
		}
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = redactValue(rv.Index(i).Interface())
		}
		return out
	}
	return orig
}
