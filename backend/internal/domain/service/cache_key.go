package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const cacheKeyPrefix = "e360"

// ErrInvalidCacheKey is returned when a key part is empty or malformed.
var ErrInvalidCacheKey = errors.New("cache: invalid key")

func validateCacheKeyPart(name, v string) error {
	if v == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrInvalidCacheKey, name)
	}
	if strings.Contains(v, ":") {
		return fmt.Errorf("%w: %s must not contain ':'", ErrInvalidCacheKey, name)
	}
	for _, r := range v {
		if unicode.IsSpace(r) {
			return fmt.Errorf("%w: %s must not contain whitespace", ErrInvalidCacheKey, name)
		}
	}
	return nil
}

// CacheKey builds a tenant-namespaced key: e360:{tenant_id}:<feature>:<key>.
// A nil tenant ID is rejected so a missing tenant can never collapse tenants
// into a shared namespace.
func CacheKey(tenantID uuid.UUID, feature, key string) (string, error) {
	if tenantID == uuid.Nil {
		return "", fmt.Errorf("%w: tenant id must not be nil", ErrInvalidCacheKey)
	}
	if err := validateCacheKeyPart("feature", feature); err != nil {
		return "", err
	}
	if err := validateCacheKeyPart("key", key); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s:%s:%s", cacheKeyPrefix, tenantID.String(), feature, key), nil
}

// GlobalCacheKey builds a key outside any tenant namespace, for pre-auth data
// where the tenant is not yet known: e360:global:<feature>:<key>.
func GlobalCacheKey(feature, key string) (string, error) {
	if err := validateCacheKeyPart("feature", feature); err != nil {
		return "", err
	}
	if err := validateCacheKeyPart("key", key); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:global:%s:%s", cacheKeyPrefix, feature, key), nil
}
