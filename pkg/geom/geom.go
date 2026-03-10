// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

// Package geom provides lightweight 2D geometry helpers used by mechanism solving.
package geom

import "math"

// Direction selects which side/criterion to prefer when choosing between two solutions.
type Direction int

// PickPoint selects which anchor point should be used as directional reference.
type PickPoint int

const (
	// Top prefers solutions above the reference point.
	Top Direction = iota
	// Left prefers solutions left of the reference point.
	Left
	// Bottom prefers solutions below the reference point.
	Bottom
	// Right prefers solutions right of the reference point.
	Right
	// Nearest picks the solution nearest the reference point.
	Nearest
	// Farthest picks the solution farthest from the reference point.
	Farthest
)

const (
	// First uses the first circle center as directional reference.
	First PickPoint = iota
	// Second uses the second circle center as directional reference.
	Second
	// Other uses the explicit compare point as directional reference.
	Other
)

// Point is a 2D Cartesian coordinate.
type Point struct {
	X float64
	Y float64
}

// Segment stores two endpoints of a line segment.
type Segment struct {
	A Point
	B Point
}

// Distance returns Euclidean distance between two points.
func Distance(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Solve computes one circle-circle intersection and selects a solution
// based on previous pose continuity or a directional criterion.
func Solve(a, b, c, d, f, g float64, dir Direction, point PickPoint, prev Point, compare Point) (Point, bool) {
	factor1 := 2*b*c*c - 2*b*d*d + 2*a*a*b + 2*a*a*d - 2*b*b*d + 2*c*c*d - 4*a*b*c - 4*a*c*d + 2*b*b*b + 2*d*d*d - 2*b*f + 2*b*g + 2*d*f - 2*d*g
	factor2Inner := 4*a*c*c*c + 4*b*d*d*d - 2*a*a*b*b - 6*a*a*c*c -
		2*a*a*d*d + 4*a*a*a*c - 2*b*b*c*c - 6*b*b*d*d + 4*b*b*b*d -
		2*c*c*d*d + 4*a*c*d*d + 4*a*b*b*c + 4*b*c*c*d +
		4*a*a*b*d - 8*a*b*c*d - a*a*a*a - b*b*b*b - c*c*c*c - d*d*d*d + 2*a*a*f +
		2*a*a*g + 2*b*b*f + 2*b*b*g + 2*c*c*f + 2*c*c*g + 2*d*d*f +
		2*d*d*g - 4*a*c*f - 4*a*c*g - 4*b*d*f - 4*b*d*g + 2*f*g -
		f*f - g*g
	factor2 := 2 * math.Abs(a-c) * math.Sqrt(factor2Inner)
	factor3 := -8*a*c - 8*b*d + 4*a*a + 4*b*b + 4*c*c + 4*d*d

	if math.IsNaN(factor2) || factor3 == 0 || math.IsInf(factor2, 0) {
		return Point{}, false
	}

	y1 := (factor1 + factor2) / factor3
	y2 := (factor1 - factor2) / factor3

	den := 2*a - 2*c
	if den == 0 {
		return Point{}, false
	}
	x1 := (-2*b*y1 + 2*d*y1 + a*a + b*b - c*c - d*d - f + g) / den
	x2 := (-2*b*y2 + 2*d*y2 + a*a + b*b - c*c - d*d - f + g) / den

	p1 := Point{X: x1, Y: y1}
	p2 := Point{X: x2, Y: y2}

	if prev.X != 0 && prev.Y != 0 {
		if Distance(p1, prev) < Distance(p2, prev) {
			return p1, true
		}
		return p2, true
	}

	xComp := compare.X
	yComp := compare.Y
	switch point {
	case First:
		xComp = a
		yComp = b
	case Second:
		xComp = c
		yComp = d
	}

	chosen := Point{}
	ok := false
	switch dir {
	case Top:
		if y1 < yComp {
			chosen = p1
			ok = true
		}
		if y2 < yComp {
			chosen = p2
			ok = true
		}
	case Bottom:
		if y1 > yComp {
			chosen = p1
			ok = true
		}
		if y2 > yComp {
			chosen = p2
			ok = true
		}
	case Left:
		if x1 < xComp {
			chosen = p1
			ok = true
		}
		if x2 < xComp {
			chosen = p2
			ok = true
		}
	case Right:
		if x1 > xComp {
			chosen = p1
			ok = true
		}
		if x2 > xComp {
			chosen = p2
			ok = true
		}
	case Nearest:
		if Distance(p1, Point{X: xComp, Y: yComp}) < Distance(p2, Point{X: xComp, Y: yComp}) {
			chosen = p1
		} else {
			chosen = p2
		}
		ok = true
	case Farthest:
		if Distance(p1, Point{X: xComp, Y: yComp}) > Distance(p2, Point{X: xComp, Y: yComp}) {
			chosen = p1
		} else {
			chosen = p2
		}
		ok = true
	}

	if !ok || math.IsNaN(chosen.X) || math.IsNaN(chosen.Y) || math.IsInf(chosen.X, 0) || math.IsInf(chosen.Y, 0) {
		return Point{}, false
	}
	return chosen, true
}

// SegmentsIntersect reports whether two non-parallel segments intersect.
func SegmentsIntersect(a, b, c, d Point) bool {
	// Fast boolean-only segment intersection:
	// avoids computing the intersection point and avoids divisions.
	rx := b.X - a.X
	ry := b.Y - a.Y
	sx := d.X - c.X
	sy := d.Y - c.Y

	den := rx*sy - ry*sx
	if math.Abs(den) < 1e-9 {
		return false
	}

	qpx := c.X - a.X
	qpy := c.Y - a.Y
	tNum := qpx*sy - qpy*sx
	uNum := qpx*ry - qpy*rx

	if den > 0 {
		return tNum >= 0 && tNum <= den && uNum >= 0 && uNum <= den
	}
	return tNum <= 0 && tNum >= den && uNum <= 0 && uNum >= den
}
