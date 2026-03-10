// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

// Package app contains the Ebiten runtime, rendering, and mode-specific loops.
package app

import (
	"fmt"
	"os"
	"time"

	"citybeestgo/pkg/config"
	"citybeestgo/pkg/evo"
)

// WriteSessionHeaders appends a session marker line to configured output files.
func WriteSessionHeaders(cfg config.Config) error {
	now := time.Now().Format("2006-01-02")
	header := fmt.Sprintf("------ Session: %s------ \n", now)

	paths := []string{cfg.FavouriteGenesFile, cfg.BestGenesFile}
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := f.WriteString(header); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

// RunHeadless continuously evolves genes and asynchronously persists new global bests.
func RunHeadless(cfg config.Config) error {
	engine := evo.NewMultiGA(cfg.ParallelGAWorkers, 0, cfg.PoolSize, cfg.DoomsdayClock, cfg.GenerationReportBatchSize)
	saver := newAsyncGeneSaver(256)
	engine.Start()
	defer engine.Stop()
	defer saver.Stop()

	current := engine.BestFit()
	for {
		best := engine.BestFit()
		if best != nil && (current == nil || !current.Equals(best)) {
			current = best
			saver.Queue(cfg.BestGenesFile, best, engine.Generation())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
