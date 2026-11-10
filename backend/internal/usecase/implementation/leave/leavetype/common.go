package leavetype

import (
	"strings"
	"unicode/utf8"

	leavecommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/leave"
)

func normalizeName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", leavecommon.Invalid("name is required")
	}
	if utf8.RuneCountInString(s) > 100 {
		return "", leavecommon.Invalid("name must not exceed 100 characters")
	}
	return s, nil
}

func normalizeCode(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return "", leavecommon.Invalid("code is required")
	}
	if utf8.RuneCountInString(s) > 30 {
		return "", leavecommon.Invalid("code must not exceed 30 characters")
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return "", leavecommon.Invalid("code may contain only letters, digits, '_' and '-'")
		}
	}
	return s, nil
}
