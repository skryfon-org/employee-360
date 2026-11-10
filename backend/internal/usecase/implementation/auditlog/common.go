// Package auditlog implements the read-only audit-log viewer usecases.
package auditlog

import (
	"encoding/json"
	"strings"
)

// sensitiveKeyParts are case-insensitive substrings; any metadata key
// containing one is dropped before the audit blob is exposed. Current writers
// store no secrets, but metadata is free-form so this is a defence in depth.
var sensitiveKeyParts = []string{
	"password", "passwd", "secret", "token", "hash", "otp",
	"authorization", "credential", "api_key", "apikey", "private_key", "cookie",
}

func isSensitiveKey(k string) bool {
	lk := strings.ToLower(k)
	for _, p := range sensitiveKeyParts {
		if strings.Contains(lk, p) {
			return true
		}
	}
	return false
}

func redact(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if isSensitiveKey(k) {
				continue
			}
			out[k] = redact(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redact(val)
		}
		return out
	default:
		return v
	}
}

// sanitizeMetadata removes sensitive keys (recursively) from a stored JSON
// blob. Empty or malformed input yields "{}"; a non-object JSON value is
// preserved after redaction of any nested objects.
func sanitizeMetadata(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return "{}"
	}
	out, err := json.Marshal(redact(v))
	if err != nil {
		return "{}"
	}
	return string(out)
}
