package entity

import (
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFormatEmployeeCode(t *testing.T) {
	cases := []struct {
		prefix  string
		padding int
		n       int64
		want    string
	}{
		{"EMP", 4, 1, "EMP0001"},
		{"EMP", 4, 42, "EMP0042"},
		{"SKRYFON", 4, 40001, "SKRYFON40001"}, // overflow past the width is never truncated
		{"EMP", 4, 9999, "EMP9999"},
		{"EMP", 4, 10000, "EMP10000"},
		{"A", 1, 7, "A7"},
		{"A", 10, 5, "A0000000005"},
		{"EMP", 2, 123456, "EMP123456"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, FormatEmployeeCode(c.prefix, c.padding, c.n))
	}
}

func TestEmployeeCodePattern(t *testing.T) {
	re := EmployeeCodePattern("EMP")
	for _, ok := range []string{"EMP1", "EMP0001", "EMP123456"} {
		assert.True(t, re.MatchString(ok), ok)
	}
	for _, bad := range []string{"EMP", "emp0001", "XEMP0001", "EMP-0001", "EMP00A1", "EMP0001 ", "AB0001"} {
		assert.False(t, re.MatchString(bad), bad)
	}
	assert.True(t, EmployeeCodePattern("A.B").MatchString("A.B1"), "prefix is matched literally")
	assert.False(t, EmployeeCodePattern("A.B").MatchString("AXB1"))
}

func TestOrganizationSettingsDefaults(t *testing.T) {
	s := DefaultOrganizationSettings(uuid.New())
	assert.Equal(t, "EMP", s.EmployeeCodePrefix)
	assert.Equal(t, 4, s.EmployeeCodePadding)
	assert.EqualValues(t, 1, s.EmployeeCodeStartNumber)
	assert.EqualValues(t, 1, s.EmployeeCodeNextNumber)
	assert.False(t, s.CodesIssued())
	assert.Equal(t, "EMP0001", s.NextEmployeeCode())
	s.EmployeeCodeLastIssuedNumber = 1
	assert.True(t, s.CodesIssued())
}

func TestParseEmployeeCodeNumber(t *testing.T) {
	n, ok := parseEmployeeCodeNumber("EMP", "EMP0042")
	assert.True(t, ok)
	assert.EqualValues(t, 42, n)
	_, ok = parseEmployeeCodeNumber("EMP", "XYZ1")
	assert.False(t, ok)
}

// parseEmployeeCodeNumber returns the numeric tail of a code matching
// EmployeeCodePattern(prefix). Test-only helper.
func parseEmployeeCodeNumber(prefix, code string) (int64, bool) {
	if !EmployeeCodePattern(prefix).MatchString(code) {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimPrefix(code, prefix), 10, 64)
	return n, err == nil
}
