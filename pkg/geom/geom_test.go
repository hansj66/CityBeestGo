package geom

import "testing"

func TestSegmentsIntersectCases(t *testing.T) {
	tests := []struct {
		name       string
		a, b, c, d Point
		want       bool
	}{
		{
			name: "crossing",
			a:    Point{X: 0, Y: 0},
			b:    Point{X: 2, Y: 2},
			c:    Point{X: 0, Y: 2},
			d:    Point{X: 2, Y: 0},
			want: true,
		},
		{
			name: "disjoint",
			a:    Point{X: 0, Y: 0},
			b:    Point{X: 1, Y: 0},
			c:    Point{X: 2, Y: 0},
			d:    Point{X: 3, Y: 0},
			want: false,
		},
		{
			name: "touching endpoint",
			a:    Point{X: 0, Y: 0},
			b:    Point{X: 2, Y: 0},
			c:    Point{X: 2, Y: 0},
			d:    Point{X: 2, Y: 2},
			want: true,
		},
		{
			name: "parallel",
			a:    Point{X: 0, Y: 0},
			b:    Point{X: 2, Y: 0},
			c:    Point{X: 0, Y: 1},
			d:    Point{X: 2, Y: 1},
			want: false,
		},
		{
			name: "collinear overlap treated as non-intersecting",
			a:    Point{X: 0, Y: 0},
			b:    Point{X: 3, Y: 0},
			c:    Point{X: 1, Y: 0},
			d:    Point{X: 2, Y: 0},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SegmentsIntersect(tc.a, tc.b, tc.c, tc.d)
			if got != tc.want {
				t.Fatalf("SegmentsIntersect() = %v, want %v", got, tc.want)
			}
		})
	}
}
