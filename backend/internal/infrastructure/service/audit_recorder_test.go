package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
)

type fakeAuditRepo struct {
	logs []*entity.AuditLog
	err  error
	ctx  context.Context
}

func (f *fakeAuditRepo) Create(c context.Context, l *entity.AuditLog) error {
	f.ctx = c
	if f.err != nil {
		return f.err
	}
	f.logs = append(f.logs, l)
	return nil
}
func (f *fakeAuditRepo) ListByTenantID(context.Context, uuid.UUID, int, int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}
func (f *fakeAuditRepo) ListByEntity(context.Context, uuid.UUID, string, uuid.UUID, int, int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func TestAuditRecorder(t *testing.T) {
	tenant, actor, ent := uuid.New(), uuid.New(), uuid.New()

	t.Run("persists entry", func(t *testing.T) {
		repo := &fakeAuditRepo{}
		type k struct{}
		c := context.WithValue(context.Background(), k{}, "tx")
		require.NoError(t, NewAuditRecorder(repo).Record(c, tenant, actor, "department.create", "department", ent, map[string]any{"name": "x"}))
		require.Len(t, repo.logs, 1)
		l := repo.logs[0]
		assert.NotEqual(t, uuid.Nil, l.ID)
		assert.Equal(t, tenant, l.TenantID)
		assert.Equal(t, &actor, l.ActorUserID)
		assert.Equal(t, "department.create", l.Action)
		assert.Equal(t, ent, l.EntityID)
		assert.False(t, l.CreatedAt.IsZero())
		assert.JSONEq(t, `{"name":"x"}`, l.Metadata)
		assert.Equal(t, "tx", repo.ctx.Value(k{}), "caller context (tx) must reach the repository")
	})

	t.Run("rejects nil ids", func(t *testing.T) {
		repo := &fakeAuditRepo{}
		r := NewAuditRecorder(repo)
		assert.ErrorIs(t, r.Record(context.Background(), uuid.Nil, actor, "a.b", "e", ent, nil), ErrAuditInvalidIdentity)
		assert.ErrorIs(t, r.Record(context.Background(), tenant, uuid.Nil, "a.b", "e", ent, nil), ErrAuditInvalidIdentity)
		assert.Empty(t, repo.logs)
	})

	t.Run("rejects bad action names", func(t *testing.T) {
		repo := &fakeAuditRepo{}
		r := NewAuditRecorder(repo)
		for _, a := range []string{"", "create", "Department.Create", "department.", ".create", "dept create"} {
			assert.ErrorIs(t, r.Record(context.Background(), tenant, actor, a, "e", ent, nil), ErrAuditInvalidAction, a)
		}
		assert.ErrorIs(t, r.Record(context.Background(), tenant, actor, "a.b", " ", ent, nil), ErrAuditInvalidEntityType)
		assert.Empty(t, repo.logs)
	})

	t.Run("redacts sensitive keys", func(t *testing.T) {
		repo := &fakeAuditRepo{}
		meta := map[string]any{
			"email": "a@b.c", "password": "p", "New_Password": "p", "refresh_token": "t", "otp_code": "1",
			"client_secret": "s", "nested": map[string]any{"token": "t", "ok": 1},
		}
		require.NoError(t, NewAuditRecorder(repo).Record(context.Background(), tenant, actor, "user.update", "user", ent, meta))
		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(repo.logs[0].Metadata), &got))
		assert.Equal(t, map[string]any{"email": "a@b.c", "nested": map[string]any{"ok": float64(1)}}, got)
		assert.Contains(t, meta, "password", "input map must not be mutated")
	})

	t.Run("propagates repository error", func(t *testing.T) {
		boom := errors.New("boom")
		err := NewAuditRecorder(&fakeAuditRepo{err: boom}).Record(context.Background(), tenant, actor, "a.b", "e", ent, nil)
		assert.ErrorIs(t, err, boom)
	})
}

func TestIsSensitive(t *testing.T) {
	sensitive := []string{
		"password", "Password", "newPassword", "New_Password", "passwd", "token", "accessToken",
		"refresh_token", "refresh-token", "otp", "otp_code", "OTPCode", "client_secret", "clientSecret",
		"authorization", "Authorization", "credential", "credentials", "user.credentials",
		"api_key", "apiKey", "api-key", "API_KEY", "apikey", "private_key", "privateKey", "private key", "PrivateKey",
	}
	for _, k := range sensitive {
		assert.True(t, isSensitive(k), k)
	}
	safe := []string{
		"footprint", "hotpath", "name", "is_active", "email", "department_id", "keyword", "tokenizer",
		"secretary", "monkey", "api", "key", "private", "author", "otpx", "",
	}
	for _, k := range safe {
		assert.False(t, isSensitive(k), k)
	}
}

func TestRedactTypes(t *testing.T) {
	nested := map[string]string{"token": "t", "name": "n"}
	list := []map[string]any{{"password": "p", "id": 1}, {"ok": true}}
	meta := map[string]any{
		"ms":    nested,
		"lm":    list,
		"sl":    map[string][]string{"otp": {"1"}, "tags": {"x"}},
		"deep":  []any{map[string]string{"api_key": "k", "v": "1"}},
		"arr":   [1]map[string]any{{"authorization": "x", "a": 1}},
		"ptr":   &nested,
		"bytes": []byte("raw"),
		"nilm":  map[string]string(nil),
	}
	out := redact(meta)
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"ms":{"name":"n"},
		"lm":[{"id":1},{"ok":true}],
		"sl":{"tags":["x"]},
		"deep":[{"v":"1"}],
		"arr":[{"a":1}],
		"ptr":{"name":"n"},
		"bytes":"cmF3",
		"nilm":null
	}`, string(raw))

	// inputs untouched
	assert.Equal(t, map[string]string{"token": "t", "name": "n"}, nested)
	assert.Equal(t, "p", list[0]["password"])
	assert.Contains(t, meta["sl"], "otp")
	assert.Contains(t, meta["deep"].([]any)[0], "api_key")
	assert.Contains(t, meta["arr"].([1]map[string]any)[0], "authorization")
}
