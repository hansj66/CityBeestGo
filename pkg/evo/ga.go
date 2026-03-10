// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

// Package evo implements the genetic algorithm engines used by the app.
package evo

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"citybeestgo/pkg/model"
)

// GA holds one evolving population and its generation-level statistics.
type GA struct {
	mu                        sync.RWMutex
	pool                      []*model.Gene
	poolSize                  int
	initialDoomsdayClock      int
	bestFit                   *model.Gene
	bestScore                 float64
	generation                int
	doomsdayClock             int
	generationsWithoutImprove int
	rng                       *rand.Rand
}

const (
	defaultPoolSize      = 5000
	defaultDoomsdayClock = 500
)

// NewGA returns a GA instance with default population and doomsday settings.
func NewGA(seed int64) *GA {
	return NewGAWithParams(seed, defaultPoolSize, defaultDoomsdayClock)
}

// NewGAWithParams returns a GA instance with explicit population and reset parameters.
func NewGAWithParams(seed int64, poolSize int, doomsdayClock int) *GA {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}
	if doomsdayClock <= 0 {
		doomsdayClock = defaultDoomsdayClock
	}

	ga := &GA{
		poolSize:             poolSize,
		initialDoomsdayClock: doomsdayClock,
		doomsdayClock:        doomsdayClock,
		rng:                  rand.New(rand.NewSource(seed)),
		bestScore:            math.Inf(-1),
	}
	ga.fillGenePool()
	if len(ga.pool) > 0 {
		ga.bestFit = ga.pool[0]
		ga.bestScore = ga.pool[0].Score()
	}
	return ga
}

// fillGenePool replaces the pool with freshly sampled viable genes.
func (g *GA) fillGenePool() {
	g.pool = g.pool[:0]
	for i := 0; i < g.poolSize; i++ {
		g.pool = append(g.pool, model.CreateViable(g.rng))
	}
}

// Step advances one generation: rank, select, breed/mutate, refill, and update stats.
func (g *GA) Step() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.doomsdayClock == 0 {
		g.fillGenePool()
		g.doomsdayClock = g.initialDoomsdayClock
	}

	sort.Slice(g.pool, func(i, j int) bool {
		return g.pool[i].Score() > g.pool[j].Score()
	})
	if len(g.pool) == 0 {
		g.fillGenePool()
		return
	}

	if g.pool[0].Score() > g.bestScore {
		g.bestFit = g.pool[0]
		g.bestScore = g.bestFit.Score()
		g.doomsdayClock = g.initialDoomsdayClock
		g.generationsWithoutImprove = 0
	} else {
		g.doomsdayClock--
		g.generationsWithoutImprove++
	}

	freshPool := make([]*model.Gene, g.poolSize)
	idx := 0
	mutationRiskLevel := g.initialDoomsdayClock - g.doomsdayClock
	if mutationRiskLevel < 0 {
		mutationRiskLevel = 0
	}
	if mutationRiskLevel > model.MutationRiskScale {
		mutationRiskLevel = model.MutationRiskScale
	}

	mostFit := minInt(150, g.poolSize)
	eliteSpawn := g.poolSize * 3 / 5
	for i := 0; i < eliteSpawn; i++ {
		freshPool[idx] = model.CreateOffspring(g.pool[g.rng.Intn(mostFit)], g.pool[g.rng.Intn(mostFit)], mutationRiskLevel, g.rng)
		idx++
	}

	randomSpawn := g.poolSize / 5
	for i := 0; i < randomSpawn; i++ {
		freshPool[idx] = model.CreateOffspring(g.pool[g.rng.Intn(g.poolSize)], g.pool[g.rng.Intn(g.poolSize)], mutationRiskLevel, g.rng)
		idx++
	}

	for i := 0; i < g.poolSize-eliteSpawn-randomSpawn; i++ {
		freshPool[idx] = model.CreateViable(g.rng)
		idx++
	}

	g.pool = freshPool
	g.generation++
}

// Generation returns the current generation counter.
func (g *GA) Generation() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.generation
}

// GenerationsWithoutImprovement returns the current stagnation counter.
func (g *GA) GenerationsWithoutImprovement() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.generationsWithoutImprove
}

// DoomsdayClock returns generations remaining before full pool reset.
func (g *GA) DoomsdayClock() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.doomsdayClock
}

// BestFit returns the best-known gene, lazily creating one if needed.
func (g *GA) BestFit() *model.Gene {
	g.mu.RLock()
	best := g.bestFit
	g.mu.RUnlock()
	if best != nil {
		return best
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.bestFit == nil {
		g.bestFit = model.CreateViable(g.rng)
		g.bestScore = g.bestFit.Score()
	}
	return g.bestFit
}

// minInt returns the smaller integer.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
