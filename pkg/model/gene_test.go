package model

import (
	"math"
	"math/rand"
	"testing"
)

func TestCreateViableGene(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	g := CreateViable(rng)
	if g == nil {
		t.Fatalf("CreateViable returned nil")
	}
	if len(g.GCurve) < 2 {
		t.Fatalf("expected non-trivial curve, got %d points", len(g.GCurve))
	}
	if math.IsNaN(g.Score()) || math.IsInf(g.Score(), 0) {
		t.Fatalf("invalid score: %v", g.Score())
	}
}
