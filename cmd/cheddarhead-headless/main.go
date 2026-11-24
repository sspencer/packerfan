package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"cheddarhead/assets"
	"cheddarhead/internal/api"
	"cheddarhead/internal/component"
	"cheddarhead/internal/sheet"
	"cheddarhead/internal/sim"
)

func main() {
	var (
		artFile    string
		msPacman   bool
		level      int
		seed       int64
		verifyFile string
		dumpMazes  bool
	)

	flag.StringVarP(&artFile, "art", "a", "", "path to game spritesheet PNG (default: embedded game-real.png else game.png)")
	flag.BoolVarP(&msPacman, "ms", "m", false, "default to Ms. Pac-Man mode for reset (default: classic Pac-Man)")
	flag.IntVarP(&level, "level", "l", 1, "default starting level for reset (1..255)")
	flag.Int64VarP(&seed, "seed", "s", component.DefaultPRNGSeed, "default PRNG seed for reset")
	flag.StringVar(&verifyFile, "verify", "", "replay a .pfr file headlessly and exit non-zero on mismatch")
	flag.BoolVar(&dumpMazes, "dump-mazes", false, "print the five parsed text mazes and hashes, then exit")
	flag.Parse()

	s, err := loadSheet(artFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	mazes := component.ParseMazeSet(s.IsPixelLit)

	if dumpMazes {
		fmt.Printf("art.file: %s\n", s.Name())
		fmt.Printf("art.sha256: %s\n", s.SHA256())
		fmt.Printf("maze_hash: %s\n", mazes.Hash())
		for i := range component.NumMazes {
			tmpl := mazes.Template(i)
			fmt.Printf("\n--- maze %d (dots=%d) ---\n%s\n", tmpl.Index, tmpl.Dots, tmpl.String())
		}
		return
	}

	if verifyFile != "" {
		rep, err := sim.ReadReplayFile(verifyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		env := sim.NewEnv(sim.Config{Mazes: mazes})
		res, err := env.Replay(rep, sim.ArtInfo{File: s.Name(), SHA256: s.SHA256()})
		if err != nil {
			fmt.Fprintf(os.Stderr, "VERIFY FAILED: %v\n", err)
			os.Exit(1)
		}
		if res.ArtSHA256Warn {
			fmt.Fprintf(os.Stderr, "warning: %s\n", res.Warning)
		}
		fmt.Printf("OK ticks=%d score=%d level=%d checkpoints=%d hash=%016x\n",
			res.Ticks, res.Score, res.Level, res.CheckpointsOK, res.FinalHash)
		return
	}

	session := api.NewSession(api.SessionConfig{
		Sheet: s,
		Mazes: mazes,
		Defaults: sim.Config{
			PacmanMode: !msPacman,
			StartLevel: max(1, min(level, 255)),
			Seed:       seed,
		},
	})

	if err := api.ServeNDJSON(os.Stdin, os.Stdout, session); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func loadSheet(artFile string) (*sheet.Sheet, error) {
	if artFile != "" {
		return sheet.LoadFile(artFile)
	}
	return sheet.Load(assets.FS, assets.GameAssetPath(assets.FS))
}
