// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

package evo

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"citybeestgo/pkg/model"
)

// MultiGA coordinates multiple GA workers and exposes a single aggregated view.
type MultiGA struct {
	workers      []*GA
	workersCount int
	poolSize     int
	batchSize    int

	mu                         sync.RWMutex
	bestFit                    *model.Gene
	bestScore                  float64
	generation                 int
	generationsWithoutImprove  int
	doomsdayClock              int
	workerGenerations          []int
	lastGenerationDurationNsec atomic.Int64

	started   atomic.Bool
	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewMultiGA builds a lock-step orchestrator over multiple independent GA workers.
func NewMultiGA(workerCount int, seed int64, poolSize int, doomsdayClock int, batchSize int) *MultiGA {
	if workerCount <= 0 {
		workerCount = 1
	}
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}
	if doomsdayClock <= 0 {
		doomsdayClock = defaultDoomsdayClock
	}
	if batchSize <= 0 {
		batchSize = 1
	}
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	m := &MultiGA{
		workers:           make([]*GA, workerCount),
		workersCount:      workerCount,
		poolSize:          poolSize,
		batchSize:         batchSize,
		bestScore:         math.Inf(-1),
		stopCh:            make(chan struct{}),
		doneCh:            make(chan struct{}),
		workerGenerations: make([]int, workerCount),
	}

	for i := 0; i < workerCount; i++ {
		workerSeed := seed + int64(i)*7919
		ga := NewGAWithParams(workerSeed, poolSize, doomsdayClock)
		m.workers[i] = ga
		m.workerGenerations[i] = ga.Generation()
		if best := ga.BestFit(); best != nil && best.Score() > m.bestScore {
			m.bestFit = best
			m.bestScore = best.Score()
			m.generationsWithoutImprove = ga.GenerationsWithoutImprovement()
			m.doomsdayClock = ga.DoomsdayClock()
		}
	}

	return m
}

// Start launches the orchestrator loop once.
func (m *MultiGA) Start() {
	m.startOnce.Do(func() {
		m.started.Store(true)
		go m.run()
	})
}

// Stop requests shutdown and waits for the orchestrator to exit once.
func (m *MultiGA) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
		if m.started.Load() {
			<-m.doneCh
		}
	})
}

// run executes synchronized worker rounds and publishes aggregated best/stats.
func (m *MultiGA) run() {
	defer close(m.doneCh)

	type workerReport struct {
		workerIdx                 int
		bestFit                   *model.Gene
		bestScore                 float64
		generation                int
		generationsWithoutImprove int
		doomsdayClock             int
	}

	for {
		select {
		case <-m.stopCh:
			return
		default:
		}

		roundStart := time.Now()
		reports := make(chan workerReport, len(m.workers))

		var wg sync.WaitGroup
		wg.Add(len(m.workers))
		for i, ga := range m.workers {
			go func(workerIdx int, workerGA *GA) {
				defer wg.Done()
				for step := 0; step < m.batchSize; step++ {
					workerGA.Step()
				}
				best := workerGA.BestFit()
				score := math.Inf(-1)
				if best != nil {
					score = best.Score()
				}
				reports <- workerReport{
					workerIdx:                 workerIdx,
					bestFit:                   best,
					bestScore:                 score,
					generation:                workerGA.Generation(),
					generationsWithoutImprove: workerGA.GenerationsWithoutImprovement(),
					doomsdayClock:             workerGA.DoomsdayClock(),
				}
			}(i, ga)
		}
		wg.Wait()
		close(reports)

		roundBestScore := math.Inf(-1)
		var roundBestFit *model.Gene
		roundBestWithoutImprove := 0
		roundBestDoomsday := 0
		roundGeneration := 0
		workerGenerations := make([]int, len(m.workers))

		for report := range reports {
			workerGenerations[report.workerIdx] = report.generation
			if report.generation > roundGeneration {
				roundGeneration = report.generation
			}
			if report.bestScore > roundBestScore {
				roundBestScore = report.bestScore
				roundBestFit = report.bestFit
				roundBestWithoutImprove = report.generationsWithoutImprove
				roundBestDoomsday = report.doomsdayClock
			}
		}

		m.mu.Lock()
		m.workerGenerations = workerGenerations
		m.generation = roundGeneration
		m.generationsWithoutImprove = roundBestWithoutImprove
		m.doomsdayClock = roundBestDoomsday
		if roundBestFit != nil && roundBestScore > m.bestScore {
			m.bestFit = roundBestFit
			m.bestScore = roundBestScore
		}
		m.mu.Unlock()

		m.lastGenerationDurationNsec.Store(time.Since(roundStart).Nanoseconds())
	}
}

// BestFit returns the globally best gene seen by the orchestrator.
func (m *MultiGA) BestFit() *model.Gene {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bestFit
}

// Generation returns the last published lock-step generation.
func (m *MultiGA) Generation() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.generation
}

// GenerationsWithoutImprovement returns the stagnation count from the current best worker.
func (m *MultiGA) GenerationsWithoutImprovement() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.generationsWithoutImprove
}

// DoomsdayClock returns the doomsday value from the current best worker.
func (m *MultiGA) DoomsdayClock() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.doomsdayClock
}

// LastGenerationDuration returns the wall-clock duration of the latest round.
func (m *MultiGA) LastGenerationDuration() time.Duration {
	return time.Duration(m.lastGenerationDurationNsec.Load())
}

// WorkerCount returns the configured number of parallel GA workers.
func (m *MultiGA) WorkerCount() int {
	return m.workersCount
}

// PoolSize returns the per-worker GA pool size.
func (m *MultiGA) PoolSize() int {
	return m.poolSize
}

// BatchSize returns local steps each worker performs per synchronized round.
func (m *MultiGA) BatchSize() int {
	return m.batchSize
}
