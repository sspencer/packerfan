package system_test

import (
	"testing"

	"github.com/yohamta/donburi"

	"cheddarhead/internal/archetype"
	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
	"cheddarhead/internal/system"
)

func TestStartGameAtLevel(t *testing.T) {
	cases := []struct {
		name          string
		pacmanMode    bool
		inputLevel    int
		wantLevel     int
		wantMazeIndex int
	}{
		{"MsPacman clamp low", false, 0, 1, 0},
		{"MsPacman negative clamp", false, -5, 1, 0},
		{"MsPacman level 1", false, 1, 1, 0},
		{"MsPacman level 3", false, 3, 3, 1},
		{"MsPacman level 6", false, 6, 6, 2},
		{"MsPacman level 10", false, 10, 10, 3},
		{"MsPacman level 14", false, 14, 14, 2},
		{"MsPacman level 18", false, 18, 18, 3},
		{"MsPacman clamp high", false, 999, 255, 2},
		{"Pacman level 7", true, 7, 7, 4},
		{"Pacman clamp high", true, 300, 255, 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := donburi.NewWorld()
			archetype.CreateGameState(w, tc.pacmanMode, 3)
			archetype.CreateMaze(w, tc.pacmanMode)
			archetype.CreatePRNG(w)
			archetype.SpawnPlayer(w)
			ghosts := archetype.SpawnAllGhosts(w)

			gs := component.MustFindGameState(w)
			gs.Score = 12340
			gs.HighScore = 50000
			gs.Lives = 1
			component.Ghost.Get(ghosts[0]).DotCounter = 15

			system.StartGameAtLevel(w, tc.inputLevel, nil)

			maze := component.MustFindMaze(w)
			if gs.Level != tc.wantLevel {
				t.Errorf("GameState.Level = %d, want %d", gs.Level, tc.wantLevel)
			}
			if maze.Level != tc.wantLevel {
				t.Errorf("Maze.Level = %d, want %d", maze.Level, tc.wantLevel)
			}
			if got := maze.GetIndex(); got != tc.wantMazeIndex {
				t.Errorf("Maze.GetIndex() = %d, want %d", got, tc.wantMazeIndex)
			}
			wantSpec := config.GetLevelSpec(tc.wantLevel)
			if gs.Spec != wantSpec {
				t.Errorf("GameState.Spec = %+v, want %+v", gs.Spec, wantSpec)
			}
			if gs.Score != 0 {
				t.Errorf("Score = %d, want 0", gs.Score)
			}
			if gs.HighScore != 50000 {
				t.Errorf("HighScore = %d, want 50000 (preserved)", gs.HighScore)
			}
			if gs.Lives != config.InitialLives {
				t.Errorf("Lives = %d, want %d", gs.Lives, config.InitialLives)
			}
			if gs.Players[0].Level != tc.wantLevel {
				t.Errorf("Players[0].Level = %d, want %d", gs.Players[0].Level, tc.wantLevel)
			}
			if got := component.Ghost.Get(ghosts[0]).DotCounter; got != 0 {
				t.Errorf("Ghost DotCounter = %d, want 0", got)
			}
		})
	}
}
