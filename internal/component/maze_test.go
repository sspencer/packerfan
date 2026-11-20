package component_test

import (
	"strings"
	"testing"

	"pacfan/internal/component"
	"pacfan/internal/config"
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
