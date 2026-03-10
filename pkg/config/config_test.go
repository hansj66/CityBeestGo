package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	// Verifies missing config files are non-fatal so first-time startup works without setup.
	path := filepath.Join(t.TempDir(), "missing.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ParallelGAWorkers != 1 {
		t.Fatalf("ParallelGAWorkers = %d, want 1", cfg.ParallelGAWorkers)
	}
	if cfg.PoolSize != 5000 {
		t.Fatalf("PoolSize = %d, want 5000", cfg.PoolSize)
	}
	if cfg.DoomsdayClock != 500 {
		t.Fatalf("DoomsdayClock = %d, want 500", cfg.DoomsdayClock)
	}
	if cfg.BestGenesFile != "fittest.txt" {
		t.Fatalf("BestGenesFile = %q, want %q", cfg.BestGenesFile, "fittest.txt")
	}
	if cfg.FavouriteGenesFile != "genes.txt" {
		t.Fatalf("FavouriteGenesFile = %q, want %q", cfg.FavouriteGenesFile, "genes.txt")
	}
	if cfg.GenerationReportBatchSize != 1 {
		t.Fatalf("GenerationReportBatchSize = %d, want 1", cfg.GenerationReportBatchSize)
	}
}

func TestLoadMalformedJSONReturnsError(t *testing.T) {
	// Verifies malformed JSON fails fast to avoid silently running with unintended settings.
	path := filepath.Join(t.TempDir(), "citybeest.json")
	if err := os.WriteFile(path, []byte(`{"parallel_ga_workers":`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadValidGAParams(t *testing.T) {
	// Verifies explicit GA parameters in config are respected for worker count, pool size, and doomsday clock.
	path := filepath.Join(t.TempDir(), "citybeest.json")
	if err := os.WriteFile(path, []byte(`{"parallel_ga_workers":4,"pool_size":6000,"doomsday_clock":750,"best_genes_file":"best_genes.txt","favourite_genes_file.txt":"picked_genes.txt","generation_report_batch_size":3}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ParallelGAWorkers != 4 {
		t.Fatalf("ParallelGAWorkers = %d, want 4", cfg.ParallelGAWorkers)
	}
	if cfg.PoolSize != 6000 {
		t.Fatalf("PoolSize = %d, want 6000", cfg.PoolSize)
	}
	if cfg.DoomsdayClock != 750 {
		t.Fatalf("DoomsdayClock = %d, want 750", cfg.DoomsdayClock)
	}
	if cfg.BestGenesFile != "best_genes.txt" {
		t.Fatalf("BestGenesFile = %q, want %q", cfg.BestGenesFile, "best_genes.txt")
	}
	if cfg.FavouriteGenesFile != "picked_genes.txt" {
		t.Fatalf("FavouriteGenesFile = %q, want %q", cfg.FavouriteGenesFile, "picked_genes.txt")
	}
	if cfg.GenerationReportBatchSize != 3 {
		t.Fatalf("GenerationReportBatchSize = %d, want 3", cfg.GenerationReportBatchSize)
	}
}

func TestLoadInvalidGAParamsFallBackToDefaults(t *testing.T) {
	// Verifies invalid non-positive GA params are clamped to safe defaults to avoid runtime instability.
	path := filepath.Join(t.TempDir(), "citybeest.json")
	if err := os.WriteFile(path, []byte(`{"parallel_ga_workers":0,"pool_size":0,"doomsday_clock":0,"best_genes_file":"","favourite_genes_file.txt":"","generation_report_batch_size":0}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ParallelGAWorkers != 1 {
		t.Fatalf("ParallelGAWorkers = %d, want 1", cfg.ParallelGAWorkers)
	}
	if cfg.PoolSize != 5000 {
		t.Fatalf("PoolSize = %d, want 5000", cfg.PoolSize)
	}
	if cfg.DoomsdayClock != 500 {
		t.Fatalf("DoomsdayClock = %d, want 500", cfg.DoomsdayClock)
	}
	if cfg.BestGenesFile != "fittest.txt" {
		t.Fatalf("BestGenesFile = %q, want %q", cfg.BestGenesFile, "fittest.txt")
	}
	if cfg.FavouriteGenesFile != "genes.txt" {
		t.Fatalf("FavouriteGenesFile = %q, want %q", cfg.FavouriteGenesFile, "genes.txt")
	}
	if cfg.GenerationReportBatchSize != 1 {
		t.Fatalf("GenerationReportBatchSize = %d, want 1", cfg.GenerationReportBatchSize)
	}
}

func TestLoadUsesDefaultPathWhenEmpty(t *testing.T) {
	// Verifies empty path resolves to ./citybeest.json so callers can rely on convention over wiring.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "citybeest.json")
	if err := os.WriteFile(path, []byte(`{"parallel_ga_workers":3,"pool_size":7000,"doomsday_clock":900,"best_genes_file":"history_snapshot.txt","favourite_genes_file.txt":"session_genes.txt","generation_report_batch_size":2}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ParallelGAWorkers != 3 {
		t.Fatalf("ParallelGAWorkers = %d, want 3", cfg.ParallelGAWorkers)
	}
	if cfg.PoolSize != 7000 {
		t.Fatalf("PoolSize = %d, want 7000", cfg.PoolSize)
	}
	if cfg.DoomsdayClock != 900 {
		t.Fatalf("DoomsdayClock = %d, want 900", cfg.DoomsdayClock)
	}
	if cfg.BestGenesFile != "history_snapshot.txt" {
		t.Fatalf("BestGenesFile = %q, want %q", cfg.BestGenesFile, "history_snapshot.txt")
	}
	if cfg.FavouriteGenesFile != "session_genes.txt" {
		t.Fatalf("FavouriteGenesFile = %q, want %q", cfg.FavouriteGenesFile, "session_genes.txt")
	}
	if cfg.GenerationReportBatchSize != 2 {
		t.Fatalf("GenerationReportBatchSize = %d, want 2", cfg.GenerationReportBatchSize)
	}
}

func TestLoadLegacyGenerationBatchSizeKey(t *testing.T) {
	// Verifies backward compatibility: old generation_batch_size key is still accepted if new key is absent.
	path := filepath.Join(t.TempDir(), "citybeest.json")
	if err := os.WriteFile(path, []byte(`{"generation_batch_size":4}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GenerationReportBatchSize != 4 {
		t.Fatalf("GenerationReportBatchSize = %d, want 4", cfg.GenerationReportBatchSize)
	}
}
