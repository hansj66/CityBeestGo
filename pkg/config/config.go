// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

// Package config loads and normalizes runtime configuration for CityBeestGo.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

const defaultConfigPath = "./citybeest.json"

// Config holds runtime-tunable settings loaded from the JSON config file.
type Config struct {
	ParallelGAWorkers         int    `json:"parallel_ga_workers"`
	PoolSize                  int    `json:"pool_size"`
	DoomsdayClock             int    `json:"doomsday_clock"`
	BestGenesFile             string `json:"best_genes_file"`
	FavouriteGenesFile        string `json:"favourite_genes_file.txt"`
	GenerationReportBatchSize int    `json:"generation_report_batch_size"`
}

// Default returns the built-in configuration used when values are not provided.
func Default() Config {
	return Config{
		ParallelGAWorkers:         1,
		PoolSize:                  5000,
		DoomsdayClock:             500,
		BestGenesFile:             "fittest.txt",
		FavouriteGenesFile:        "genes.txt",
		GenerationReportBatchSize: 1,
	}
}

// Load reads configuration from path (or ./citybeest.json when path is empty),
// merges it into defaults, and normalizes invalid values back to safe defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var loaded struct {
		ParallelGAWorkers         int    `json:"parallel_ga_workers"`
		PoolSize                  int    `json:"pool_size"`
		DoomsdayClock             int    `json:"doomsday_clock"`
		BestGenesFile             string `json:"best_genes_file"`
		FavouriteGenesFile        string `json:"favourite_genes_file.txt"`
		GenerationReportBatchSize int    `json:"generation_report_batch_size"`
	}
	if err := json.Unmarshal(data, &loaded); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.ParallelGAWorkers = loaded.ParallelGAWorkers
	cfg.PoolSize = loaded.PoolSize
	cfg.DoomsdayClock = loaded.DoomsdayClock
	cfg.BestGenesFile = loaded.BestGenesFile
	cfg.FavouriteGenesFile = loaded.FavouriteGenesFile
	cfg.GenerationReportBatchSize = loaded.GenerationReportBatchSize
	if cfg.GenerationReportBatchSize <= 0 {
		cfg.GenerationReportBatchSize = 1
	}

	if cfg.ParallelGAWorkers <= 0 {
		cfg.ParallelGAWorkers = 1
	}
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 5000
	}
	if cfg.DoomsdayClock <= 0 {
		cfg.DoomsdayClock = 500
	}
	if cfg.BestGenesFile == "" {
		cfg.BestGenesFile = "fittest.txt"
	}
	if cfg.FavouriteGenesFile == "" {
		cfg.FavouriteGenesFile = "genes.txt"
	}
	if cfg.GenerationReportBatchSize <= 0 {
		cfg.GenerationReportBatchSize = 1
	}

	return cfg, nil
}
