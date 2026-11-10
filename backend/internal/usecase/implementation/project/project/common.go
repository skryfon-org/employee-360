// Package project implements the project usecases. All
// repository calls take the explicit tenantID resolved by the handler from the
// auth context; nothing here reads identity from context.Context.
package project

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/skryfon-org/employee-360/backend/internal/domain/entity"
	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	"github.com/skryfon-org/employee-360/backend/internal/domain/repository"
	projectcommon "github.com/skryfon-org/employee-360/backend/internal/usecase/implementation/project"
)

// Validation sentinels (each matches domainerrors.ErrInvalidProjectData).
var (
	ErrNameRequired       = projectcommon.Required("name")
	ErrNameTooLong        = projectcommon.TooLong("name", 150)
	ErrCodeRequired       = projectcommon.Required("code")
	ErrCodeTooLong        = projectcommon.TooLong("code", 30)
	ErrCodeInvalid        = projectcommon.Invalid("code: may contain only letters, digits, '-' and '_'")
	ErrDescriptionTooLong = projectcommon.TooLong("description", 2000)
	ErrClientNameTooLong  = projectcommon.TooLong("client_name", 150)
	ErrStatusInvalid      = projectcommon.Invalid("status: must be one of " + strings.Join(entity.ProjectStatuses, ", "))
	ErrProjectDateOrder   = projectcommon.Invalid("end_date: must not be before start_date")
)

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]*$`)

func validStatus(s string) bool {
	for _, v := range entity.ProjectStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// normalizeCode trims and upper-cases a project code, then validates it.
func normalizeCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	switch {
	case code == "":
		return "", ErrCodeRequired
	case utf8.RuneCountInString(code) > 30:
		return "", ErrCodeTooLong
	case !codePattern.MatchString(code):
		return "", ErrCodeInvalid
	}
	return code, nil
}

func normalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		return "", ErrNameRequired
	case utf8.RuneCountInString(name) > 150:
		return "", ErrNameTooLong
	}
	return name, nil
}

func normalizeDescription(raw string) (string, error) {
	d := strings.TrimSpace(raw)
	if utf8.RuneCountInString(d) > 2000 {
		return "", ErrDescriptionTooLong
	}
	return d, nil
}

const (
	maxCodeLen      = 30
	fallbackCode    = "PROJECT"
	maxCodeAttempts = 50
)

// asciiFold maps common accented Latin letters to their ASCII base; anything
// else non-ASCII is treated as a separator.
var asciiFold = strings.NewReplacer(
	"À", "A", "Á", "A", "Â", "A", "Ã", "A", "Ä", "A", "Å", "A", "Æ", "AE", "Ç", "C",
	"È", "E", "É", "E", "Ê", "E", "Ë", "E", "Ì", "I", "Í", "I", "Î", "I", "Ï", "I",
	"Ñ", "N", "Ò", "O", "Ó", "O", "Ô", "O", "Õ", "O", "Ö", "O", "Ø", "O",
	"Ù", "U", "Ú", "U", "Û", "U", "Ü", "U", "Ý", "Y", "ß", "SS",
	"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "æ", "ae", "ç", "c",
	"è", "e", "é", "e", "ê", "e", "ë", "e", "ì", "i", "í", "i", "î", "i", "ï", "i",
	"ñ", "n", "ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o",
	"ù", "u", "ú", "u", "û", "u", "ü", "u", "ý", "y", "ÿ", "y",
)

// baseCodeFromName derives the unsuffixed code: upper-case ASCII alphanumeric
// words joined by '-', cut at a word boundary (or mid-word if the first word
// alone is too long) to fit limit runes. Falls back to "PROJECT".
func baseCodeFromName(name string, limit int) string {
	words := strings.FieldsFunc(strings.ToUpper(asciiFold.Replace(name)), func(r rune) bool {
		return !(r < unicode.MaxASCII && (unicode.IsDigit(r) || unicode.IsUpper(r)))
	})
	if len(words) == 0 {
		return fallbackCode
	}
	out := words[0]
	if len(out) > limit {
		return out[:limit]
	}
	for _, w := range words[1:] {
		if len(out)+1+len(w) > limit {
			break
		}
		out += "-" + w
	}
	return out
}

// candidateCode returns base for n<=1, else base-n, truncating base so the
// result stays within maxCodeLen.
func candidateCode(name string, n int) string {
	if n <= 1 {
		return baseCodeFromName(name, maxCodeLen)
	}
	suffix := "-" + strconv.Itoa(n)
	base := baseCodeFromName(name, maxCodeLen-len(suffix))
	return base + suffix
}

// firstFreeCode returns the first candidate code (starting at suffix index
// start; 1 = unsuffixed) not yet taken in the tenant, plus the index used.
func firstFreeCode(c context.Context, repo repository.ProjectRepository, tenantID uuid.UUID, name string, start int) (string, int, error) {
	if start < 1 {
		start = 1
	}
	for n := start; n < start+maxCodeAttempts*20; n++ {
		code := candidateCode(name, n)
		taken, err := repo.ExistsByCode(c, tenantID, code)
		if err != nil {
			return "", 0, err
		}
		if !taken {
			return code, n, nil
		}
	}
	return "", 0, domainerrors.ErrProjectCodeTaken
}
