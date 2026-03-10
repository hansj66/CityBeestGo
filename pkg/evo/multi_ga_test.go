// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

package evo

import (
	"testing"
	"time"
)

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool, failureMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(failureMsg)
}

func allEqual(values []int) bool {
	if len(values) == 0 {
		return true
	}
	first := values[0]
	for _, v := range values[1:] {
		if v != first {
			return false
		}
	}
	return true
}

func copyInts(src []int) []int {
	dst := make([]int, len(src))
	copy(dst, src)
	return dst
}

func snapshotWorkerGenerations(m *MultiGA) []int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return copyInts(m.workerGenerations)
}

// Verifies the orchestrator lifecycle is safe: start progresses work and stop is deterministic and idempotent.
func TestMultiGAStartStopLifecycle(t *testing.T) {
	m := NewMultiGA(2, 123, 5000, 500, 1)
	m.Start()

	waitUntil(t, 30*time.Second, func() bool {
		return m.Generation() > 0
	}, "generation did not advance after Start()")

	m.Stop()
	m.Stop()
}

// Verifies parallel workers produce a best-of-best snapshot and publish round timing once execution starts.
func TestMultiGAPublishesBestAndTiming(t *testing.T) {
	m := NewMultiGA(3, 456, 5000, 500, 1)
	m.Start()
	defer m.Stop()

	waitUntil(t, 30*time.Second, func() bool {
		return m.Generation() > 0
	}, "generation did not advance")

	best := m.BestFit()
	if best == nil {
		t.Fatal("BestFit() = nil, want non-nil")
	}
	if m.LastGenerationDuration() <= 0 {
		t.Fatalf("LastGenerationDuration() = %v, want > 0", m.LastGenerationDuration())
	}
}

// Verifies workers stay in lock step so each global generation represents one completed round across all workers.
func TestMultiGALockStepGeneration(t *testing.T) {
	m := NewMultiGA(4, 789, 5000, 500, 1)
	m.Start()
	defer m.Stop()

	waitUntil(t, 30*time.Second, func() bool {
		if m.Generation() == 0 {
			return false
		}
		workerGens := snapshotWorkerGenerations(m)
		return len(workerGens) == 4 && allEqual(workerGens) && workerGens[0] == m.Generation()
	}, "workers did not stay in lock-step generations")
}

// Verifies single-worker mode still progresses so multi-worker support does not regress baseline behavior.
func TestMultiGASingleWorkerProgresses(t *testing.T) {
	m := NewMultiGA(1, 999, 5000, 500, 1)
	m.Start()
	defer m.Stop()

	waitUntil(t, 30*time.Second, func() bool {
		return m.Generation() > 0
	}, "single-worker generation did not advance")

	if m.BestFit() == nil {
		t.Fatal("BestFit() = nil, want non-nil in single-worker mode")
	}
}
