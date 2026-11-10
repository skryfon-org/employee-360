// Package pagination defines the single paging convention shared by every
// paginated list endpoint: query params page/page_size, defaults 1/20, hard
// cap 100. Non-integer values are rejected by the HTTP layer (400); integer
// values that are out of range are clamped (page < 1 -> 1, page_size < 1 ->
// default, page_size > max -> max).
package pagination

const (
	// DefaultPage is the page served when none is requested.
	DefaultPage = 1
	// DefaultPageSize is the page size used when none (or < 1) is requested.
	DefaultPageSize = 20
	// MaxPageSize is the largest page size ever served.
	MaxPageSize = 100
)

// Normalize clamps a requested page/page_size into the supported range and
// returns the effective values plus the row offset.
func Normalize(page, pageSize int) (p, size, offset int) {
	p, size = page, pageSize
	if p < 1 {
		p = DefaultPage
	}
	if size < 1 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return p, size, (p - 1) * size
}

// TotalPages returns the number of pages needed for total items.
func TotalPages(total int64, pageSize int) int {
	if pageSize < 1 || total <= 0 {
		return 0
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}
