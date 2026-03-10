// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

// Package model defines the core gene representation, kinematic simulation,
// scoring, persistence, and history loading for CityBeestGo.
package model

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"citybeestgo/pkg/geom"
)

// Gene contains one full linkage parameter set and cached scoring values.
type Gene struct {
	Ax float64
	Ay float64
	Cx float64
	Cy float64
	AB float64
	CD float64
	BD float64
	CH float64
	BH float64
	CF float64
	DF float64
	FE float64
	HE float64
	EG float64
	HG float64

	// GCurve is the sampled tip trajectory over one full crank cycle.
	GCurve []geom.Point

	ScoreValue                   float64
	LoopOpeningScore             float64
	GroundContactScore           float64
	StanceLevelnessScore         float64
	StrideLengthScore            float64
	SharpEdgePenaltyScore        float64
	SelfIntersectionPenaltyScore float64
}

// MutationRiskScale is the random range used for per-parameter mutation checks.
// A mutation risk level of N corresponds to an approximate mutation chance of N/MutationRiskScale.
const MutationRiskScale = 1000

// Pose is one solved mechanism pose for a given crank angle.
type Pose struct {
	B geom.Point
	D geom.Point
	H geom.Point
	F geom.Point
	E geom.Point
	G geom.Point
}

// Score returns the cached total score for this gene.
func (g *Gene) Score() float64 {
	return g.ScoreValue
}

// Equals reports whether two genes have identical parameters and traced curve.
func (g *Gene) Equals(other *Gene) bool {
	if g == nil || other == nil {
		return g == other
	}
	if g.Ax != other.Ax || g.Ay != other.Ay || g.Cx != other.Cx || g.Cy != other.Cy ||
		g.AB != other.AB || g.CD != other.CD || g.BD != other.BD || g.CH != other.CH || g.BH != other.BH ||
		g.CF != other.CF || g.DF != other.DF || g.FE != other.FE || g.HE != other.HE || g.EG != other.EG || g.HG != other.HG {
		return false
	}
	if len(g.GCurve) != len(other.GCurve) {
		return false
	}
	for i := range g.GCurve {
		if g.GCurve[i] != other.GCurve[i] {
			return false
		}
	}
	return true
}

// Save appends this gene as one tab-separated line to the given file path.
func (g *Gene) Save(path string, generation int) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%d\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\t%.12g\n",
		generation, g.ScoreValue, g.Ax, g.Ay, g.Cx, g.Cy, g.AB, g.CD, g.BD, g.CH, g.BH, g.CF, g.DF, g.FE, g.HE, g.EG, g.HG)
	return err
}

func pseudoRandom(rng *rand.Rand) float64 {
	return 10 + rng.Float64()*190
}

// CreateViable keeps sampling randomized linkage parameters until CreateGene
// returns a mechanically valid, score-ready gene.
func CreateViable(rng *rand.Rand) *Gene {
	ax := 600.0
	ay := 100.0
	for {
		cx := ax - pseudoRandom(rng)
		cy := ay - pseudoRandom(rng)
		if rng.Intn(100) < 50 {
			cy = ay + pseudoRandom(rng)
		}

		ab := pseudoRandom(rng)
		cd := pseudoRandom(rng)
		bd := pseudoRandom(rng)
		ch := pseudoRandom(rng)
		bh := pseudoRandom(rng)
		cf := pseudoRandom(rng)
		df := pseudoRandom(rng)
		fe := pseudoRandom(rng)
		he := pseudoRandom(rng)
		eg := pseudoRandom(rng)
		hg := pseudoRandom(rng)

		gene, ok := CreateGene(ax, ay, cx, cy, ab, cd, bd, ch, bh, cf, df, fe, he, eg, hg)
		if ok {
			return gene
		}
	}
}

// CreateOffspring mixes two parent genes, applies mutationRiskLevel-scaled mutation,
// retries until a viable mechanism is produced, and falls back to a fresh viable gene.
func CreateOffspring(mama, papa *Gene, mutationRiskLevel int, rng *rand.Rand) *Gene {
	for i := 0; i < 5000; i++ {
		var ax, ay, cx, cy, ab, cd, bd, ch, bh, cf, df, fe, he, eg, hg float64

		if rng.Intn(100) < 50 {
			if rng.Intn(100) < 50 {
				ax = mama.Ax
			} else {
				ax = papa.Ax
			}
			if rng.Intn(100) < 50 {
				ay = mama.Ay
			} else {
				ay = papa.Ay
			}
			if rng.Intn(100) < 50 {
				cx = mama.Cx
			} else {
				cx = papa.Cx
			}
			if rng.Intn(100) < 50 {
				cy = mama.Cy
			} else {
				cy = papa.Cy
			}
			if rng.Intn(100) < 50 {
				ab = mama.AB
			} else {
				ab = papa.AB
			}
			if rng.Intn(100) < 50 {
				bd = mama.BD
			} else {
				bd = papa.BD
			}
			if rng.Intn(100) < 50 {
				bh = mama.BH
			} else {
				bh = papa.BH
			}
			if rng.Intn(100) < 50 {
				cf = mama.CF
			} else {
				cf = papa.CF
			}
			if rng.Intn(100) < 50 {
				df = mama.DF
			} else {
				df = papa.DF
			}
			if rng.Intn(100) < 50 {
				cd = mama.CD
			} else {
				cd = papa.CD
			}
			if rng.Intn(100) < 50 {
				ch = mama.CH
			} else {
				ch = papa.CH
			}
			if rng.Intn(100) < 50 {
				fe = mama.FE
			} else {
				fe = papa.FE
			}
			if rng.Intn(100) < 50 {
				he = mama.HE
			} else {
				he = papa.HE
			}
			if rng.Intn(100) < 50 {
				eg = mama.EG
			} else {
				eg = papa.EG
			}
			if rng.Intn(100) < 50 {
				hg = mama.HG
			} else {
				hg = papa.HG
			}
		} else {
			// Alternative crossover mode: copy one coherent parameter block from one parent
			// and the complementary block from the other, instead of per-parameter coin flips.
			if rng.Intn(100) < 50 {
				ax, ay, cx, cy = mama.Ax, mama.Ay, mama.Cx, mama.Cy
				ab, bd, bh, cf, df, cd = mama.AB, mama.BD, mama.BH, mama.CF, mama.DF, mama.CD
				ch, fe, he, eg, hg = papa.CH, papa.FE, papa.HE, papa.EG, papa.HG
			} else {
				ax, ay, cx, cy = papa.Ax, papa.Ay, papa.Cx, papa.Cy
				ab, bd, bh, cf, df, cd = papa.AB, papa.BD, papa.BH, papa.CF, papa.DF, papa.CD
				ch, fe, he, eg, hg = mama.CH, mama.FE, mama.HE, mama.EG, mama.HG
			}
		}

		// Mutation chance is roughly mutationRiskLevel/MutationRiskScale per parameter.
		// Callers can raise mutationRiskLevel as stagnation increases.
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			cx = (cx + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			cy = (cy + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			ab = (ab + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			cd = (cd + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			bd = (bd + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			ch = (ch + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			bh = (bh + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			cf = (cf + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			df = (df + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			fe = (fe + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			he = (he + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			eg = (eg + pseudoRandom(rng)) / 2
		}
		if rng.Intn(MutationRiskScale) <= mutationRiskLevel {
			hg = (hg + pseudoRandom(rng)) / 2
		}

		gene, ok := CreateGene(ax, ay, cx, cy, ab, cd, bd, ch, bh, cf, df, fe, he, eg, hg)
		if ok {
			return gene
		}
	}
	return CreateViable(rng)
}

// PoseAt solves linkage geometry at a crank angle and returns the resulting pose.
func (g *Gene) PoseAt(angle float64, prev Pose) (Pose, bool) {
	bx := g.AB*math.Sin(angle) + g.Ax
	by := g.AB*math.Cos(angle) + g.Ay

	d, ok := geom.Solve(g.Cx, g.Cy, bx, by, g.CD*g.CD, g.BD*g.BD, geom.Top, geom.Other, prev.D, geom.Point{X: g.Cx, Y: g.Cy})
	if !ok {
		return Pose{}, false
	}
	h, ok := geom.Solve(g.Cx, g.Cy, bx, by, g.CH*g.CH, g.BH*g.BH, geom.Bottom, geom.Other, prev.H, geom.Point{X: g.Cx, Y: g.Cy})
	if !ok {
		return Pose{}, false
	}
	f, ok := geom.Solve(g.Cx, g.Cy, d.X, d.Y, g.CF*g.CF, g.DF*g.DF, geom.Left, geom.Other, prev.F, geom.Point{X: g.Cx, Y: g.Cy})
	if !ok {
		return Pose{}, false
	}
	e, ok := geom.Solve(f.X, f.Y, h.X, h.Y, g.FE*g.FE, g.HE*g.HE, geom.Left, geom.Other, prev.E, h)
	if !ok {
		return Pose{}, false
	}
	gp, ok := geom.Solve(e.X, e.Y, h.X, h.Y, g.EG*g.EG, g.HG*g.HG, geom.Bottom, geom.Other, prev.G, h)
	if !ok {
		return Pose{}, false
	}

	return Pose{
		B: geom.Point{X: bx, Y: by},
		D: d,
		H: h,
		F: f,
		E: e,
		G: gp,
	}, true
}

// CreateGene simulates one full crank cycle for the provided linkage parameters
// and returns a scored gene only if the mechanism stays valid for the whole cycle.
func CreateGene(ax, ay, cx, cy, ab, cd, bd, ch, bh, cf, df, fe, he, eg, hg float64) (*Gene, bool) {
	viable := false
	steps := 0
	angle := 6.28
	curve := make([]geom.Point, 0, 130)
	yMin := math.MaxFloat64
	kinematics := Gene{
		Ax: ax, Ay: ay, Cx: cx, Cy: cy,
		AB: ab, CD: cd, BD: bd, CH: ch, BH: bh, CF: cf, DF: df, FE: fe, HE: he, EG: eg, HG: hg,
	}
	prev := Pose{}
	// Sweep angle through one revolution, validating each pose.
	for angle >= 0 {
		pose, ok := kinematics.PoseAt(angle, prev)
		if !ok {
			viable = false
			break
		}
		prev = pose
		viable = true

		pA := geom.Point{X: ax, Y: ay}
		pB := pose.B
		pC := geom.Point{X: cx, Y: cy}
		pD := pose.D
		pE := pose.E
		pF := pose.F
		pG := pose.G
		pH := pose.H

		// Reject mechanisms with forbidden self/cross intersections.
		if hasForbiddenLinkIntersections(pA, pB, pC, pD, pE, pF, pG, pH) {
			viable = false
			break
		}

		// Reject mechanisms that drift from fixed rod-length constraints.
		if math.Abs(geom.Distance(pA, pB)-ab) > 0.1 ||
			math.Abs(geom.Distance(pB, pD)-bd) > 0.1 ||
			math.Abs(geom.Distance(pB, pH)-bh) > 0.1 ||
			math.Abs(geom.Distance(pD, pC)-cd) > 0.1 ||
			math.Abs(geom.Distance(pC, pH)-ch) > 0.1 ||
			math.Abs(geom.Distance(pD, pF)-df) > 0.1 ||
			math.Abs(geom.Distance(pC, pF)-cf) > 0.1 ||
			math.Abs(geom.Distance(pF, pE)-fe) > 0.1 ||
			math.Abs(geom.Distance(pE, pG)-eg) > 0.1 ||
			math.Abs(geom.Distance(pH, pG)-hg) > 0.1 ||
			math.Abs(geom.Distance(pE, pH)-he) > 0.1 {
			viable = false
			break
		}

		// Record foot trajectory for later scoring and drawing.
		curve = append(curve, pG)
		angle -= 0.05
		steps++

		if pG.Y < yMin {
			yMin = pG.Y
		}
		// Early-reject if the traced curve dips below the E-point envelope.
		if yMin < pE.Y {
			return nil, false
		}
	}

	// Accept only fully valid full-cycle solutions, close the curve, then score.
	if viable && steps == 126 {
		if len(curve) > 0 {
			curve = append(curve, curve[0])
		}
		g := &Gene{
			Ax: ax, Ay: ay, Cx: cx, Cy: cy,
			AB: ab, CD: cd, BD: bd, CH: ch, BH: bh, CF: cf, DF: df, FE: fe, HE: he, EG: eg, HG: hg,
			GCurve: curve,
		}
		g.CalculateScore()
		return g, true
	}
	return nil, false
}

// hasForbiddenLinkIntersections reports whether the mechanism links intersect
// in disallowed pairings for a single solved pose.
func hasForbiddenLinkIntersections(a, b, c, d, e, f, g, h geom.Point) bool {
	if geom.SegmentsIntersect(c, a, d, f) { // CA-DF
		return true
	}
	if geom.SegmentsIntersect(c, a, f, e) { // CA-FE
		return true
	}
	if geom.SegmentsIntersect(c, a, e, h) { // CA-HE
		return true
	}
	if geom.SegmentsIntersect(c, a, e, g) { // CA-EG
		return true
	}
	if geom.SegmentsIntersect(c, a, h, g) { // CA-HG
		return true
	}
	if geom.SegmentsIntersect(a, b, c, h) { // AB-CH
		return true
	}
	if geom.SegmentsIntersect(a, b, d, c) { // AB-DC
		return true
	}
	if geom.SegmentsIntersect(a, b, d, f) { // AB-DF
		return true
	}
	if geom.SegmentsIntersect(a, b, c, f) { // AB-CF
		return true
	}
	if geom.SegmentsIntersect(a, b, f, e) { // AB-FE
		return true
	}
	if geom.SegmentsIntersect(a, b, e, g) { // AB-EG
		return true
	}
	if geom.SegmentsIntersect(a, b, h, g) { // AB-HG
		return true
	}
	if geom.SegmentsIntersect(a, b, e, h) { // AB-HE
		return true
	}
	if geom.SegmentsIntersect(b, d, c, h) { // BD-CH
		return true
	}
	if geom.SegmentsIntersect(b, d, c, f) { // BD-CF
		return true
	}
	if geom.SegmentsIntersect(b, d, f, e) { // BD-FE
		return true
	}
	if geom.SegmentsIntersect(b, d, e, g) { // BD-EG
		return true
	}
	if geom.SegmentsIntersect(b, d, h, g) { // BD-HG
		return true
	}
	if geom.SegmentsIntersect(b, d, e, h) { // BD-HE
		return true
	}
	if geom.SegmentsIntersect(b, h, d, c) { // BH-DC
		return true
	}
	if geom.SegmentsIntersect(b, h, d, f) { // BH-DF
		return true
	}
	if geom.SegmentsIntersect(b, h, c, f) { // BH-CF
		return true
	}
	if geom.SegmentsIntersect(b, h, f, e) { // BH-FE
		return true
	}
	if geom.SegmentsIntersect(b, h, e, g) { // BH-EG
		return true
	}
	if geom.SegmentsIntersect(d, c, f, e) { // DC-FE
		return true
	}
	if geom.SegmentsIntersect(d, c, e, g) { // DC-EG
		return true
	}
	if geom.SegmentsIntersect(d, c, h, g) { // DC-HG
		return true
	}
	if geom.SegmentsIntersect(c, h, d, f) { // CH-DF
		return true
	}
	if geom.SegmentsIntersect(c, h, f, e) { // CH-FE
		return true
	}
	if geom.SegmentsIntersect(c, h, e, g) { // CH-EG
		return true
	}
	if geom.SegmentsIntersect(d, f, e, g) { // DF-EG
		return true
	}
	if geom.SegmentsIntersect(d, f, h, g) { // DF-HG
		return true
	}
	if geom.SegmentsIntersect(c, f, e, g) { // CF-EG
		return true
	}
	if geom.SegmentsIntersect(c, f, h, g) { // CF-HG
		return true
	}
	if geom.SegmentsIntersect(f, e, h, g) { // FE-HG
		return true
	}
	if geom.SegmentsIntersect(f, e, c, h) { // FE-CH
		return true
	}
	return false
}

// segmentVerticalIntersection returns where segment p1->p2 crosses x=const within [yMin, yMax].
func segmentVerticalIntersection(p1, p2 geom.Point, x, yMin, yMax float64) (geom.Point, bool) {
	dx := p2.X - p1.X
	if math.Abs(dx) < 1e-9 {
		return geom.Point{}, false
	}
	t := (x - p1.X) / dx
	if t < 0 || t > 1 {
		return geom.Point{}, false
	}
	y := p1.Y + t*(p2.Y-p1.Y)
	if y < yMin || y > yMax {
		return geom.Point{}, false
	}
	return geom.Point{X: x, Y: y}, true
}

// CalculateScore computes and stores heuristic component scores and total ScoreValue.
func (g *Gene) CalculateScore() {
	g.SharpEdgePenaltyScore = 0
	g.SelfIntersectionPenaltyScore = 0

	// Score is computed from the traced foot path (GCurve) using four heuristics:
	// 1) LoopOpeningScore: favors an open loop shape (vertical gap at two interior x-slices).
	// 2) StanceLevelnessScore: favors a level bottom stroke between those slices.
	// 3) GroundContactScore: rewards how many sampled points lie on that bottom stroke.
	// 4) StrideLengthScore: favors wider-than-tall trajectories.
	minX := math.MaxFloat64
	maxX := -math.MaxFloat64
	minY := math.MaxFloat64
	maxY := -math.MaxFloat64

	for _, p := range g.GCurve {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	segmentSize := (maxX - minX) / 5
	lowerX := minX + segmentSize
	upperX := maxX - segmentSize

	// Intersect the curve with two vertical sampling lines (20% and 80% across width).
	// These crossings define the "opening" and the candidate bottom stroke level.
	lowerYs := make([]geom.Point, 0, 4)
	upperYs := make([]geom.Point, 0, 4)
	for i := 0; i < len(g.GCurve)-1; i++ {
		p1 := g.GCurve[i]
		p2 := g.GCurve[i+1]
		if p, ok := segmentVerticalIntersection(p1, p2, lowerX, minY, maxY); ok {
			lowerYs = append(lowerYs, p)
		}
		if p, ok := segmentVerticalIntersection(p1, p2, upperX, minY, maxY); ok {
			upperYs = append(upperYs, p)
		}
	}

	g.LoopOpeningScore = 0
	if len(lowerYs) == 2 && len(upperYs) == 2 {
		// Open form heuristic: larger top/bottom separation at both slices is better.
		g.LoopOpeningScore = math.Abs(lowerYs[0].Y-lowerYs[1].Y)*2 + math.Abs(upperYs[0].Y-upperYs[1].Y)
	}
	if g.LoopOpeningScore < 10 {
		// Strong penalty when the loop is too closed/narrow.
		g.LoopOpeningScore = -1000
	}

	if len(lowerYs) < 2 || len(upperYs) < 2 {
		// Without two crossings on each slice, we cannot estimate flatness/ground reliably.
		g.StanceLevelnessScore = -1000
		g.GroundContactScore = -1000
		g.StrideLengthScore = -math.Abs(maxY - minY)
		g.ScoreValue = 2*g.LoopOpeningScore + 2*g.GroundContactScore + g.StanceLevelnessScore + g.StrideLengthScore
		return
	}

	if len(lowerYs) >= 2 && lowerYs[0].Y > lowerYs[1].Y {
		lowerYs[0], lowerYs[1] = lowerYs[1], lowerYs[0]
	}
	if len(upperYs) >= 2 && upperYs[0].Y > upperYs[1].Y {
		upperYs[0], upperYs[1] = upperYs[1], upperYs[0]
	}
	// Flatness compares the lower crossing level at left/right slices.
	// Smaller difference => flatter bottom stroke => higher score.
	g.StanceLevelnessScore = 100 - 100*math.Abs(lowerYs[1].Y-upperYs[1].Y)

	// GroundContactScore counts how many sampled foot points lie near that bottom stroke
	// (between lowerX and upperX). This rewards a longer, sustained "on-ground" phase.
	g.GroundContactScore = 0
	for _, p := range g.GCurve {
		if p.X > lowerX && p.X < upperX {
			if math.Abs(p.Y-lowerYs[1].Y) < 0.1 {
				g.GroundContactScore++
			}
		}
	}
	if g.GroundContactScore < 10 {
		// Penalize curves that barely spend any samples on the inferred ground line.
		g.GroundContactScore = -1000
	}

	// Prefer long horizontal stride over tall vertical motion.
	if math.Abs(maxX-minX) > math.Abs(maxY-minY) {
		g.StrideLengthScore = math.Abs(maxX - minX)
	} else {
		g.StrideLengthScore = -math.Abs(maxY - minY)
	}

	baseScore := 2*g.LoopOpeningScore + 2*g.GroundContactScore + g.StanceLevelnessScore + g.StrideLengthScore

	// BEGIN EXPERIMENTAL PENALTIES
	// 1) Penalize sharp turn angles in the traced foot trajectory.
	sharpEdgePenalty := 0.0
	const sharpTurnDotThreshold = 0.5 // dot < 0.5 => turn angle > 60 degrees
	for i := 1; i < len(g.GCurve)-1; i++ {
		prev := g.GCurve[i-1]
		cur := g.GCurve[i]
		next := g.GCurve[i+1]

		v1x, v1y := cur.X-prev.X, cur.Y-prev.Y
		v2x, v2y := next.X-cur.X, next.Y-cur.Y
		len1 := math.Hypot(v1x, v1y)
		len2 := math.Hypot(v2x, v2y)
		if len1 < 1e-9 || len2 < 1e-9 {
			continue
		}

		dot := (v1x*v2x + v1y*v2y) / (len1 * len2)
		if dot < sharpTurnDotThreshold {
			// Larger turns (lower dot) get stronger penalty.
			sharpEdgePenalty += (sharpTurnDotThreshold - dot) * 50
		}
	}

	// 2) Apply a steep penalty for each self-intersection in the curve (e.g., figure-8).
	selfIntersectionPenalty := 0.0
	const selfIntersectionPenaltyPerCrossing = 10000.0
	segmentCount := len(g.GCurve) - 1
	for i := 0; i < segmentCount; i++ {
		a1 := g.GCurve[i]
		a2 := g.GCurve[i+1]
		for j := i + 1; j < segmentCount; j++ {
			// Skip adjacent segments and wrap-adjacent pair that share endpoints.
			if j == i+1 || (i == 0 && j == segmentCount-1) {
				continue
			}
			b1 := g.GCurve[j]
			b2 := g.GCurve[j+1]
			if geom.SegmentsIntersect(a1, a2, b1, b2) {
				selfIntersectionPenalty += selfIntersectionPenaltyPerCrossing
			}
		}
	}
	// END EXPERIMENTAL PENALTIES

	g.SharpEdgePenaltyScore = sharpEdgePenalty
	g.SelfIntersectionPenaltyScore = selfIntersectionPenalty
	g.ScoreValue = baseScore - sharpEdgePenalty - selfIntersectionPenalty
}

// HistoryGene couples a saved generation number with its reconstructed gene.
type HistoryGene struct {
	Generation int
	Gene       *Gene
}

// LoadHistory reads a saved history file and returns valid genes sorted by score.
func LoadHistory(path string) ([]*HistoryGene, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	genes := make([]*HistoryGene, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 17 {
			continue
		}
		generation, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		vals := make([]float64, 0, 15)
		for i := 2; i < 17; i++ {
			v, err := strconv.ParseFloat(parts[i], 64)
			if err != nil {
				vals = nil
				break
			}
			vals = append(vals, v)
		}
		if len(vals) != 15 {
			continue
		}
		g, ok := CreateGene(vals[0], vals[1], vals[2], vals[3], vals[4], vals[5], vals[6], vals[7], vals[8], vals[9], vals[10], vals[11], vals[12], vals[13], vals[14])
		if !ok {
			continue
		}
		genes = append(genes, &HistoryGene{Generation: generation, Gene: g})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(genes, func(i, j int) bool {
		return genes[i].Gene.Score() < genes[j].Gene.Score()
	})
	return genes, nil
}
