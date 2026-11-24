package component_test

import (
	"slices"
	"strings"
	"testing"

	"cheddarhead/assets"
	"cheddarhead/internal/component"
	"cheddarhead/internal/config"
	"cheddarhead/internal/sheet"
)

func TestCellAndMazeStringRepresentations(t *testing.T) {
	tests := []struct {
		cell       component.Cell
		wantChar   string
		wantPretty string
	}{
		{component.Empty, " ", "   "},
		{component.Wall, "#", "███"},
		{component.Door, "-", "───"},
		{component.Dot, ".", " · "},
		{component.Power, "O", " ○ "},
		{component.Tunnel, "@", "@@@"},
	}

	for _, tt := range tests {
		if got := tt.cell.String(); got != tt.wantChar {
			t.Errorf("Cell(%d).String() = %q, want %q", tt.cell, got, tt.wantChar)
		}
		if got := tt.cell.PrettyString(); got != tt.wantPretty {
			t.Errorf("Cell(%d).PrettyString() = %q, want %q", tt.cell, got, tt.wantPretty)
		}
	}

	m := component.NewMazeData(true)
	m.SetCell(0, 0, component.Wall)
	m.SetCell(1, 0, component.Dot)
	m.SetCell(2, 0, component.Power)
	m.SetCell(3, 0, component.Door)
	m.SetCell(4, 0, component.Tunnel)

	rows := m.Rows()
	if len(rows) != config.GameHeight {
		t.Fatalf("len(Rows()) = %d, want %d", len(rows), config.GameHeight)
	}
	for y, r := range rows {
		if len(r) != config.GameWidth {
			t.Fatalf("row %d length = %d, want %d", y, len(r), config.GameWidth)
		}
	}
	if !strings.HasPrefix(rows[0], "#.O-@ ") {
		t.Fatalf("row 0 prefix = %q, want prefix %q", rows[0][:6], "#.O-@ ")
	}

	compact := m.String()
	lines := strings.Split(compact, "\n")
	if len(lines) != config.GameHeight || lines[0] != rows[0] {
		t.Fatalf("MazeData.String() did not match Rows() (lines=%d)", len(lines))
	}

	pretty := m.PrettyString()
	if !strings.Contains(pretty, "███") || !strings.Contains(pretty, " 0│") {
		t.Fatalf("MazeData.PrettyString() missing expected Unicode grid/headers")
	}
}

func TestMazeTemplateParityAndTextRoundTrip(t *testing.T) {
	for _, assetPath := range []string{assets.GameAssetFallbackPath, assets.GameAssetPath(assets.FS)} {
		t.Run(assetPath, func(t *testing.T) {
			s, err := sheet.Load(assets.FS, assetPath)
			if err != nil {
				t.Fatalf("sheet.Load(%q): %v", assetPath, err)
			}

			set := component.ParseMazeSet(s.IsPixelLit)
			if len(set.Hash()) != 64 {
				t.Fatalf("MazeSet.Hash() length = %d, want 64", len(set.Hash()))
			}

			// Verify all 5 mazes: template-backed LoadLevel == pixel-sampled LoadLevel.
			cases := []struct {
				pacmanMode bool
				level      int
				wantIndex  int
			}{
				{false, 1, 0},
				{false, 3, 1},
				{false, 6, 2},
				{false, 10, 3},
				{true, 1, 4},
			}

			for _, tc := range cases {
				pixelMaze := component.NewMazeData(tc.pacmanMode)
				pixelDots := pixelMaze.LoadLevel(tc.level, s.IsPixelLit)

				tmplMaze := component.NewMazeData(tc.pacmanMode)
				tmplMaze.UseTemplates(set)
				// Pass a panicking isPixelLit to prove template-backed LoadLevel never calls it.
				tmplDots := tmplMaze.LoadLevel(tc.level, func(int, int) bool {
					t.Fatalf("template-backed LoadLevel unexpectedly called isPixelLit")
					return false
				})

				if tmplMaze.GetIndex() != tc.wantIndex {
					t.Fatalf("maze index = %d, want %d", tmplMaze.GetIndex(), tc.wantIndex)
				}
				if tmplDots != pixelDots {
					t.Errorf("maze %d dots: template=%d pixel=%d", tc.wantIndex, tmplDots, pixelDots)
				}
				if tmplMaze.GhostHouse() != pixelMaze.GhostHouse() {
					t.Errorf("maze %d GhostHouse: template=%+v pixel=%+v", tc.wantIndex, tmplMaze.GhostHouse(), pixelMaze.GhostHouse())
				}
				if !slices.Equal(tmplMaze.Rows(), pixelMaze.Rows()) {
					t.Errorf("maze %d rows differ between template and pixel LoadLevel", tc.wantIndex)
				}

				// Text round-trip: Rows() -> ParseMazeTemplateFromRows -> identical grid/dots/house.
				tmpl := set.Template(tc.wantIndex)
				rows := tmpl.Rows()
				parsed, err := component.ParseMazeTemplateFromRows(tc.wantIndex, rows)
				if err != nil {
					t.Fatalf("ParseMazeTemplateFromRows(%d): %v", tc.wantIndex, err)
				}
				if parsed.Grid != tmpl.Grid {
					t.Errorf("maze %d round-trip grid differs", tc.wantIndex)
				}
				if parsed.Dots != tmpl.Dots {
					t.Errorf("maze %d round-trip dots = %d, want %d", tc.wantIndex, parsed.Dots, tmpl.Dots)
				}
				if parsed.House != tmpl.House {
					t.Errorf("maze %d round-trip house = %+v, want %+v", tc.wantIndex, parsed.House, tmpl.House)
				}
			}

			// Mutating a single tile changes MazeSet.Hash().
			modified := set.Templates
			modified[0].Grid[0][0] = component.Empty
			set2 := component.NewMazeSetFromTemplates(modified)
			if set2.Hash() == set.Hash() {
				t.Errorf("expected MazeSet.Hash() to change after modifying a tile")
			}
		})
	}
}
