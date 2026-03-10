// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

// Package app contains the Ebiten runtime, rendering, and mode-specific loops.
package app

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"citybeestgo/pkg/config"
	"citybeestgo/pkg/evo"
	"citybeestgo/pkg/geom"
	"citybeestgo/pkg/model"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// ScreenWidth is the window width in pixels.
	ScreenWidth = 1000
	// ScreenHeight is the window height in pixels.
	ScreenHeight     = 700
	renderingOffsetX = 50
	renderingOffsetY = 150
)

// Mode selects which runtime behavior the Game executes.
type Mode int

const (
	// ModeEvolve runs live genetic evolution and rendering.
	ModeEvolve Mode = iota
	// ModeHistory replays previously saved genes.
	ModeHistory
)

// Game is the main Ebiten game state for evolve/history modes.
type Game struct {
	mode Mode

	engine             *evo.MultiGA
	saver              *asyncGeneSaver
	bestGenesFile      string
	favouriteGenesFile string
	gene               *model.Gene
	mu                 sync.Mutex

	evolveStop chan struct{}
	evolveDone chan struct{}
	stopOnce   sync.Once

	evolveGeneration                int
	evolveGenerationsWithoutImprove int
	evolveInitialDoomsdayClock      int
	evolveDoomsdayClock             int
	evolveParallelGAWorkers         int
	evolvePoolSize                  int
	evolveBatchSize                 int
	evolveGenerationTimeNanos       atomic.Int64

	history []*model.HistoryGene
	current int

	angle float64
	pose  model.Pose
	pair  model.Pose

	rng *rand.Rand
}

// NewEvolutionGame creates an evolve-mode game backed by the multi-GA engine.
func NewEvolutionGame(seed int64, cfg config.Config) *Game {
	engine := evo.NewMultiGA(cfg.ParallelGAWorkers, seed, cfg.PoolSize, cfg.DoomsdayClock, cfg.GenerationReportBatchSize)
	engine.Start()
	g := &Game{
		mode:                       ModeEvolve,
		engine:                     engine,
		saver:                      newAsyncGeneSaver(256),
		bestGenesFile:              cfg.BestGenesFile,
		favouriteGenesFile:         cfg.FavouriteGenesFile,
		gene:                       engine.BestFit(),
		angle:                      6.28,
		rng:                        rand.New(rand.NewSource(time.Now().UnixNano())),
		evolveStop:                 make(chan struct{}),
		evolveDone:                 make(chan struct{}),
		evolveParallelGAWorkers:    engine.WorkerCount(),
		evolvePoolSize:             engine.PoolSize(),
		evolveBatchSize:            engine.BatchSize(),
		evolveInitialDoomsdayClock: cfg.DoomsdayClock,
	}
	g.evolveGeneration = engine.Generation()
	g.evolveGenerationsWithoutImprove = engine.GenerationsWithoutImprovement()
	g.evolveDoomsdayClock = engine.DoomsdayClock()
	g.startEvolutionLoop()
	return g
}

// NewHistoryGame creates a replay-mode game for the provided history entries.
func NewHistoryGame(history []*model.HistoryGene) *Game {
	g := &Game{
		mode:    ModeHistory,
		history: history,
		angle:   6.28,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if len(history) > 0 {
		g.gene = history[0].Gene
	}
	return g
}

// Update advances animation and handles input for the active mode.
func (g *Game) Update() error {

	g.advanceAnimationStep()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		g.stopEvolutionLoop()
		return ebiten.Termination
	}

	switch g.mode {
	case ModeEvolve:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.mu.Lock()
			g.gene = model.CreateViable(g.rng)
			g.angle = 6.28
			g.pose = model.Pose{}
			g.pair = model.Pose{}
			g.mu.Unlock()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			gene := g.engine.BestFit()
			generation := g.engine.Generation()
			if gene != nil {
				g.saver.Queue(g.favouriteGenesFile, gene, generation)
			}
		}
	case ModeHistory:
		if len(g.history) == 0 {
			break
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
			g.mu.Lock()
			g.current++
			if g.current >= len(g.history) {
				g.current = 0
			}
			g.gene = g.history[g.current].Gene
			g.angle = 6.28
			g.pose = model.Pose{}
			g.pair = model.Pose{}
			g.mu.Unlock()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
			g.mu.Lock()
			g.current--
			if g.current < 0 {
				g.current = len(g.history) - 1
			}
			g.gene = g.history[g.current].Gene
			g.angle = 6.28
			g.pose = model.Pose{}
			g.pair = model.Pose{}
			g.mu.Unlock()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			_ = g.history[g.current].Gene.Save("selected.txt", g.history[g.current].Generation)
		}
	}

	return nil
}

// Draw renders HUD information, linkage geometry, and trajectory curves.
func (g *Game) Draw(screen *ebiten.Image) {
	g.mu.Lock()
	defer g.mu.Unlock()

	screen.Fill(color.RGBA{128, 128, 128, 0})
	if g.gene == nil {
		ebitenutil.DebugPrintAt(screen, "No gene loaded", 50, 50)
		return
	}

	lines := make([]string, 0, 32)
	if g.mode == ModeEvolve {
		generationDuration := time.Duration(g.evolveGenerationTimeNanos.Load())
		generationTimeMs := float64(generationDuration) / float64(time.Millisecond)
		genesPerSecond := 0.0
		mutationRiskLevel := g.evolveInitialDoomsdayClock - g.evolveDoomsdayClock
		mutationRisk := float64(mutationRiskLevel) / float64(model.MutationRiskScale)
		if mutationRisk < 0 {
			mutationRisk = 0
		}
		if mutationRisk > 1 {
			mutationRisk = 1
		}
		if generationDuration > 0 {
			genesPerRound := float64(g.evolveParallelGAWorkers * g.evolvePoolSize * g.evolveBatchSize)
			genesPerSecond = genesPerRound / generationDuration.Seconds()
		}
		lines = append(lines,
			fmt.Sprintf("Parallel GAs: %d", g.evolveParallelGAWorkers),
			fmt.Sprintf("GA pool size: %d", g.evolvePoolSize),
			fmt.Sprintf("Generation reporting batch size: %d", g.evolveBatchSize),
			fmt.Sprintf("Generation: %d", g.evolveGeneration),
			fmt.Sprintf("Generation time: %.0f ms", generationTimeMs),
			fmt.Sprintf("Genes / second: %.0f", genesPerSecond),
			fmt.Sprintf("No improvement in: %d", g.evolveGenerationsWithoutImprove),
			fmt.Sprintf("Doomsday in: %d generations", g.evolveDoomsdayClock),
			fmt.Sprintf("Mutation risk: %.1f%%", mutationRisk*100),
			"",
		)
	} else {
		lines = append(lines,
			fmt.Sprintf("Solution %d of %d", g.current+1, len(g.history)),
			fmt.Sprintf("Generation: %d", g.history[g.current].Generation),
			fmt.Sprintf("Score: %.2f", g.gene.Score()),
		)
	}

	lines = append(lines,
		"Evaluation metrics",
		fmt.Sprintf(". Stance Levelness: %.2f", g.gene.StanceLevelnessScore),
		fmt.Sprintf(". Loop Opening: %.2f", g.gene.LoopOpeningScore),
		fmt.Sprintf(". Ground contact: %.2f", g.gene.GroundContactScore),
		fmt.Sprintf(". Stride Length: %.2f", g.gene.StrideLengthScore),
		fmt.Sprintf(". Sharp-edge penalty (exp): -%.2f", g.gene.SharpEdgePenaltyScore),
		fmt.Sprintf(". Self-intersection penalty (exp): -%.2f", g.gene.SelfIntersectionPenaltyScore),
		fmt.Sprintf("Total Score: %.2f", g.gene.Score()),
		"",
		fmt.Sprintf("Ax %.2f", g.gene.Ax),
		fmt.Sprintf("Ay %.2f", g.gene.Ay),
		fmt.Sprintf("Cx %.2f", g.gene.Cx),
		fmt.Sprintf("Cy %.2f", g.gene.Cy),
		fmt.Sprintf("AB %.2f", g.gene.AB),
		fmt.Sprintf("CD %.2f", g.gene.CD),
		fmt.Sprintf("BD %.2f", g.gene.BD),
		fmt.Sprintf("CH %.2f", g.gene.CH),
		fmt.Sprintf("BH %.2f", g.gene.BH),
		fmt.Sprintf("CF %.2f", g.gene.CF),
		fmt.Sprintf("DF %.2f", g.gene.DF),
		fmt.Sprintf("FE %.2f", g.gene.FE),
		fmt.Sprintf("HE %.2f", g.gene.HE),
		fmt.Sprintf("EG %.2f", g.gene.EG),
		fmt.Sprintf("HG %.2f", g.gene.HG),
	)

	y := 50
	for _, line := range lines {
		ebitenutil.DebugPrintAt(screen, line, 50, y)
		y += 16
	}

	g.drawMechanism(screen, renderingOffsetX, renderingOffsetY)
	g.drawCurve(screen, renderingOffsetX, renderingOffsetY)
}

// advanceAnimationStep progresses crank angle and updates both primary/mirrored poses.
func (g *Game) advanceAnimationStep() {
	if g.gene == nil {
		return
	}
	if g.angle < 0 {
		g.angle = 6.28
		g.pose = model.Pose{}
		g.pair = model.Pose{}
	}
	pose, ok := g.gene.PoseAt(g.angle, g.pose)
	if ok {
		g.pose = pose
	}
	// Use (-angle) for the mirrored leg drive angle so that, after x-mirroring,
	// the crank remains clockwise on screen and in phase with the primary leg.
	pairAngle := math.Mod(-g.angle, 2*math.Pi)
	if pairAngle < 0 {
		pairAngle += 2 * math.Pi
	}
	pairPose, pairOK := g.gene.PoseAt(pairAngle, g.pair)
	if pairOK {
		g.pair = pairPose
	}
	g.angle -= 0.1
}

// drawMechanism renders the current linkage geometry, including the mirrored pair.
func (g *Game) drawMechanism(screen *ebiten.Image, offsetX float64, offsetY float64) {
	a := geom.Point{X: g.gene.Ax, Y: g.gene.Ay}
	c := geom.Point{X: g.gene.Cx, Y: g.gene.Cy}
	b := g.pose.B
	d := g.pose.D
	h := g.pose.H
	f := g.pose.F
	e := g.pose.E
	gp := g.pose.G

	var col color.Color = color.White
	draw := func(p1, p2 geom.Point, label string) {
		if math.IsNaN(p1.X) || math.IsNaN(p1.Y) || math.IsNaN(p2.X) || math.IsNaN(p2.Y) {
			return
		}
		vector.StrokeLine(
			screen,
			float32(p1.X+offsetX), float32(p1.Y+offsetY),
			float32(p2.X+offsetX), float32(p2.Y+offsetY),
			1,
			col,
			true,
		)
		midX := int((p1.X+p2.X)/2 + offsetX)
		midY := int((p1.Y+p2.Y)/2 + offsetY)
		ebitenutil.DebugPrintAt(screen, label, midX+4, midY-6)
	}

	draw(a, b, "ab")
	draw(a, c, "ac")
	draw(c, d, "cd")
	draw(b, d, "bd")
	draw(c, h, "ch")
	draw(b, h, "bh")
	draw(d, f, "df")
	draw(c, f, "cf")
	draw(f, e, "fe")
	draw(h, e, "he")
	draw(e, gp, "eg")
	draw(h, gp, "hg")

	// Second leg: pose is solved from a mirrored drive angle and rendered over A's vertical axis.
	mirrorX := func(p geom.Point) geom.Point {
		return geom.Point{X: 2*a.X - p.X, Y: p.Y}
	}
	ap := mirrorX(a)
	cp := mirrorX(c)
	bp := mirrorX(g.pair.B)
	dp := mirrorX(g.pair.D)
	hp := mirrorX(g.pair.H)
	fp := mirrorX(g.pair.F)
	ep := mirrorX(g.pair.E)
	gpp := mirrorX(g.pair.G)
	pairCol := color.RGBA{R: 160, G: 220, B: 255, A: 255}
	col = pairCol
	draw(ap, bp, "")
	draw(ap, cp, "")
	draw(cp, dp, "")
	draw(bp, dp, "")
	draw(cp, hp, "")
	draw(bp, hp, "")
	draw(dp, fp, "")
	draw(cp, fp, "")
	draw(fp, ep, "")
	draw(hp, ep, "")
	draw(ep, gpp, "")
	draw(hp, gpp, "")
	col = color.White

	fixedPointColor := color.RGBA{R: 255, A: 255}
	vector.FillCircle(screen, float32(a.X+offsetX), float32(a.Y+offsetY), 3, fixedPointColor, true)
	vector.FillCircle(screen, float32(c.X+offsetX), float32(c.Y+offsetY), 3, fixedPointColor, true)
	vector.FillCircle(screen, float32(cp.X+offsetX), float32(cp.Y+offsetY), 3, fixedPointColor, true)
	ebitenutil.DebugPrintAt(screen, "A", int(a.X+offsetX)+5, int(a.Y+offsetY)-10)
	ebitenutil.DebugPrintAt(screen, "C", int(c.X+offsetX)+5, int(c.Y+offsetY)-10)
	ebitenutil.DebugPrintAt(screen, "C2", int(cp.X+offsetX)+5, int(cp.Y+offsetY)-10)
}

// drawCurve renders the foot trajectory for both the base and mirrored linkage.
func (g *Game) drawCurve(screen *ebiten.Image, offsetX float64, offsetY float64) {
	mirrorX := func(p geom.Point) geom.Point {
		return geom.Point{X: 2*g.gene.Ax - p.X, Y: p.Y}
	}

	for i := 0; i < len(g.gene.GCurve)-1; i++ {
		p1 := g.gene.GCurve[i]
		p2 := g.gene.GCurve[i+1]
		vector.StrokeLine(
			screen,
			float32(p1.X+offsetX), float32(p1.Y+offsetY),
			float32(p2.X+offsetX), float32(p2.Y+offsetY),
			1,
			color.RGBA{B: 200, A: 255},
			true,
		)

		mp1 := mirrorX(p1)
		mp2 := mirrorX(p2)
		vector.StrokeLine(
			screen,
			float32(mp1.X+offsetX), float32(mp1.Y+offsetY),
			float32(mp2.X+offsetX), float32(mp2.Y+offsetY),
			1,
			color.RGBA{R: 160, G: 220, B: 255, A: 255},
			true,
		)
	}
}

// Layout reports the fixed Ebiten framebuffer size.
func (g *Game) Layout(_, _ int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// startEvolutionLoop polls engine state, applies best-fit updates, and queues persistence.
func (g *Game) startEvolutionLoop() {
	if g.mode != ModeEvolve || g.engine == nil {
		return
	}
	go func() {
		lastBest := g.engine.BestFit()
		pollTicker := time.NewTicker(20 * time.Millisecond)
		defer pollTicker.Stop()
		defer close(g.evolveDone)

		g.syncEvolutionStats()

		for {
			select {
			case <-g.evolveStop:
				return
			case <-pollTicker.C:
				best := g.engine.BestFit()
				if best != nil && (lastBest == nil || !lastBest.Equals(best)) {
					generation := g.engine.Generation()
					g.mu.Lock()
					g.gene = best
					g.angle = 6.28
					g.pose = model.Pose{}
					g.pair = model.Pose{}
					g.evolveGeneration = generation
					g.evolveGenerationsWithoutImprove = g.engine.GenerationsWithoutImprovement()
					g.evolveDoomsdayClock = g.engine.DoomsdayClock()
					g.mu.Unlock()
					g.saver.Queue(g.bestGenesFile, best, generation)
					lastBest = best
				}
				g.syncEvolutionStats()
			}
		}
	}()
}

// stopEvolutionLoop stops background polling and releases engine/saver resources once.
func (g *Game) stopEvolutionLoop() {
	if g.mode != ModeEvolve || g.evolveStop == nil || g.evolveDone == nil {
		return
	}
	g.stopOnce.Do(func() {
		close(g.evolveStop)
		<-g.evolveDone
		if g.engine != nil {
			g.engine.Stop()
		}
		if g.saver != nil {
			g.saver.Stop()
		}
	})
}

// syncEvolutionStats copies latest engine counters/timing into HUD-visible fields.
func (g *Game) syncEvolutionStats() {
	generation := g.engine.Generation()
	withoutImprove := g.engine.GenerationsWithoutImprovement()
	doomsday := g.engine.DoomsdayClock()
	duration := g.engine.LastGenerationDuration()

	g.mu.Lock()
	g.evolveGeneration = generation
	g.evolveGenerationsWithoutImprove = withoutImprove
	g.evolveDoomsdayClock = doomsday
	g.mu.Unlock()
	g.evolveGenerationTimeNanos.Store(duration.Nanoseconds())
}
