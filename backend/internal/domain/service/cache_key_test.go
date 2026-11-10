package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheKey(t *testing.T) {
	tid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	k, err := CacheKey(tid, "ratelimit", "login")
	require.NoError(t, err)
	assert.Equal(t, "e360:11111111-1111-1111-1111-111111111111:ratelimit:login", k)
}

func TestCacheKey_DiffersAcrossTenants(t *testing.T) {
	a, err := CacheKey(uuid.New(), "otp", "user@example.com")
	require.NoError(t, err)
	b, err := CacheKey(uuid.New(), "otp", "user@example.com")
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestCacheKey_Rejects(t *testing.T) {
	tid := uuid.New()
	cases := []struct {
		name, feature, key string
		tenant             uuid.UUID
	}{
		{"nil tenant", "f", "k", uuid.Nil},
		{"empty feature", "", "k", tid},
		{"empty key", "f", "", tid},
		{"colon in feature", "a:b", "k", tid},
		{"colon in key", "f", "a:b", tid},
		{"space in key", "f", "a b", tid},
		{"tab in feature", "f\t", "k", tid},
		{"newline in key", "f", "a\nb", tid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := CacheKey(c.tenant, c.feature, c.key)
			require.ErrorIs(t, err, ErrInvalidCacheKey)
		})
	}
}

func TestGlobalCacheKey(t *testing.T) {
	k, err := GlobalCacheKey("otp", "abc")
	require.NoError(t, err)
	assert.Equal(t, "e360:global:otp:abc", k)

	_, err = GlobalCacheKey("", "abc")
	require.ErrorIs(t, err, ErrInvalidCacheKey)
	_, err = GlobalCacheKey("otp", "a b")
	require.ErrorIs(t, err, ErrInvalidCacheKey)
	_, err = GlobalCacheKey("o:tp", "abc")
	require.ErrorIs(t, err, ErrInvalidCacheKey)
}

func TestCacheKey_GlobalNeverCollidesWithTenant(t *testing.T) {
	tk, err := CacheKey(uuid.New(), "otp", "x")
	require.NoError(t, err)
	gk, err := GlobalCacheKey("otp", "x")
	require.NoError(t, err)
	assert.NotEqual(t, tk, gk)
}
