package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCountLeaveDays(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	sat := []int{0, 6}
	cases := []struct {
		name, part, start, end string
		off                    []int
		want                   float64
	}{
		{"single weekday", LeaveDayPartFull, "2026-11-02", "2026-11-02", sat, 1},
		{"Thu..Mon skips weekend", LeaveDayPartFull, "2026-11-26", "2026-11-30", sat, 3},
		{"range on off days only", LeaveDayPartFull, "2026-11-28", "2026-11-29", sat, 0},
		{"single off day", LeaveDayPartFull, "2026-11-28", "2026-11-28", sat, 0},
		{"starts on off day", LeaveDayPartFull, "2026-11-28", "2026-11-30", sat, 1},
		{"half day working", LeaveDayPartFirstHalf, "2026-11-02", "2026-11-02", sat, 0.5},
		{"half day on off day", LeaveDayPartSecondHalf, "2026-11-29", "2026-11-29", sat, 0},
		{"empty part is full", "", "2026-11-02", "2026-11-03", sat, 2},
		{"custom Fri+Sat off", LeaveDayPartFull, "2026-11-26", "2026-11-30", []int{5, 6}, 3},
		{"custom Friday off, Friday single", LeaveDayPartFull, "2026-11-27", "2026-11-27", []int{5}, 0},
		{"no off days", LeaveDayPartFull, "2026-11-26", "2026-11-30", nil, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, CountLeaveDays(d(c.start), d(c.end), c.part, c.off, nil))
		})
	}
}

func TestCountLeaveDays_Holidays(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	sat := []int{0, 6}
	cases := []struct {
		name, part, start, end string
		hol                    []time.Time
		want                   float64
	}{
		{"holiday mid-range", LeaveDayPartFull, "2026-11-02", "2026-11-04", []time.Time{d("2026-11-03")}, 2},
		{"holiday on weekend not double counted", LeaveDayPartFull, "2026-11-26", "2026-11-30", []time.Time{d("2026-11-28")}, 3},
		{"only holiday + weekend", LeaveDayPartFull, "2026-11-27", "2026-11-29", []time.Time{d("2026-11-27")}, 0},
		{"half day on holiday", LeaveDayPartFirstHalf, "2026-11-03", "2026-11-03", []time.Time{d("2026-11-03")}, 0},
		{"holiday with time-of-day", LeaveDayPartFull, "2026-11-02", "2026-11-03", []time.Time{d("2026-11-03").Add(5 * time.Hour)}, 1},
		{"holiday outside range ignored", LeaveDayPartFull, "2026-11-02", "2026-11-03", []time.Time{d("2026-12-25")}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, CountLeaveDays(d(c.start), d(c.end), c.part, sat, c.hol))
		})
	}
}
