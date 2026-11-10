package pagination

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct{ p, s, wp, ws, wo int }{
		{0, 0, 1, 20, 0},
		{-3, -1, 1, 20, 0},
		{3, 10, 3, 10, 20},
		{2, 1000, 2, 100, 100},
	}
	for _, c := range cases {
		p, s, o := Normalize(c.p, c.s)
		if p != c.wp || s != c.ws || o != c.wo {
			t.Errorf("Normalize(%d,%d)=%d,%d,%d want %d,%d,%d", c.p, c.s, p, s, o, c.wp, c.ws, c.wo)
		}
	}
}

func TestTotalPages(t *testing.T) {
	if TotalPages(0, 20) != 0 || TotalPages(1, 20) != 1 || TotalPages(41, 20) != 3 || TotalPages(5, 0) != 0 {
		t.Fatal("unexpected TotalPages")
	}
}
