// Command citybeest starts CityBeestGo in evolve, headless, or history mode.
package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"citybeestgo/pkg/app"
	"citybeestgo/pkg/config"
	"citybeestgo/pkg/model"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {

	os.Setenv("EBITEN_GRAPHICS_LIBRARY", "opengl")

	configPath, args, err := splitConfigArg(os.Args[1:])
	if err != nil {
		log.Fatalf("invalid args: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := app.WriteSessionHeaders(cfg); err != nil {
		log.Fatalf("failed to write session headers: %v", err)
	}

	if len(args) == 1 && args[0] == "headless" {
		slog.Info("Running in headless mode", "savefile", cfg.BestGenesFile)
		if err := app.RunHeadless(cfg); err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(args) == 1 {
		if args[0] == "history" {
			runHistory(cfg.BestGenesFile, 100, 100)
			return
		}

	}

	x := 100
	y := 100
	if len(args) == 2 {
		if px, err := strconv.Atoi(args[0]); err == nil {
			x = px
		}
		if py, err := strconv.Atoi(args[1]); err == nil {
			y = py
		}
	}
	slog.Info("Starting evolution mode. Please wait while we initialize the population.")
	slog.Info("Genetic Algorithm", "parallell workers", cfg.ParallelGAWorkers, "single GA population size", cfg.PoolSize)

	runEvolution(x, y, cfg)
}

func runEvolution(x, y int, cfg config.Config) {
	ebiten.SetWindowSize(app.ScreenWidth, app.ScreenHeight)
	ebiten.SetWindowTitle("CityBeestGo - Evolution")
	ebiten.SetWindowPosition(x, y)
	fmt.Println("running window mode")
	if err := ebiten.RunGame(app.NewEvolutionGame(0, cfg)); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}

func splitConfigArg(args []string) (string, []string, error) {
	configPath := ""
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-config" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("missing value for -config")
			}
			configPath = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "-config=") {
			configPath = strings.TrimPrefix(arg, "-config=")
			continue
		}
		remaining = append(remaining, arg)
	}

	return configPath, remaining, nil
}

func runHistory(path string, x, y int) {
	slog.Info("Running history mode")
	history, err := model.LoadHistory(path)
	if err != nil {
		log.Fatalf("failed to load history: %v", err)
	}
	if len(history) == 0 {
		log.Fatalf("no valid genes found in history file: %s", path)
	}
	ebiten.SetWindowSize(app.ScreenWidth, app.ScreenHeight)
	ebiten.SetWindowTitle("CityBeestGo - History")
	ebiten.SetWindowPosition(x, y)
	fmt.Println("running viewer mode")
	if err := ebiten.RunGame(app.NewHistoryGame(history)); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
