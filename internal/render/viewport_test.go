package render_test

import (
	"math"
	"testing"

	"cheddarhead/internal/render"
)

func almostEqual(a, b float32) bool {
	return math.Abs(float64(a-b)) < 1e-4
}

func TestFitViewport(t *testing.T) {
	tests := []struct {
		name    string
		screenW int
		screenH int
		maxZoom float32
		want    render.Viewport
	}{
		{
			name:    "1080p landscape fills full height and centers horizontally",
			screenW: 1920,
			screenH: 1080,
			maxZoom: 0,
			want: render.Viewport{
				X:      540,
				Y:      0,
				Width:  840,
				Height: 1080,
				Scale:  3.75,
			},
		},
		{
			name:    "1440p landscape uses exact 5x scale",
			screenW: 2560,
			screenH: 1440,
			maxZoom: 0,
			want: render.Viewport{
				X:      720,
				Y:      0,
				Width:  1120,
				Height: 1440,
				Scale:  5.0,
			},
		},
		{
			name:    "4K landscape fills full height at 7.5x",
			screenW: 3840,
			screenH: 2160,
			maxZoom: 0,
			want: render.Viewport{
				X:      1080,
				Y:      0,
				Width:  1680,
				Height: 2160,
				Scale:  7.5,
			},
		},
		{
			name:    "MacBook 1440x900 fills full height",
			screenW: 1440,
			screenH: 900,
			maxZoom: 0,
			want: render.Viewport{
				X:      370,
				Y:      0,
				Width:  700,
				Height: 900,
				Scale:  3.125,
			},
		},
		{
			name:    "portrait screen fills full width and centers vertically",
			screenW: 672,
			screenH: 1200,
			maxZoom: 0,
			want: render.Viewport{
				X:      0,
				Y:      168,
				Width:  672,
				Height: 864,
				Scale:  3.0,
			},
		},
		{
			name:    "exact 3x windowed mode has zero margins",
			screenW: 672,
			screenH: 864,
			maxZoom: 0,
			want: render.Viewport{
				X:      0,
				Y:      0,
				Width:  672,
				Height: 864,
				Scale:  3.0,
			},
		},
		{
			name:    "explicit maxZoom caps scale and centers on both axes",
			screenW: 1920,
			screenH: 1080,
			maxZoom: 2,
			want: render.Viewport{
				X:      736,
				Y:      252,
				Width:  448,
				Height: 576,
				Scale:  2.0,
			},
		},
		{
			name:    "explicit maxZoom larger than screen still shrinks to fit",
			screenW: 1920,
			screenH: 1080,
			maxZoom: 10,
			want: render.Viewport{
				X:      540,
				Y:      0,
				Width:  840,
				Height: 1080,
				Scale:  3.75,
			},
		},
		{
			name:    "invalid zero dimensions fall back to native 1x",
			screenW: 0,
			screenH: 0,
			maxZoom: 0,
			want: render.Viewport{
				X:      0,
				Y:      0,
				Width:  render.NativeWidth,
				Height: render.NativeHeight,
				Scale:  1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := render.FitViewport(tt.screenW, tt.screenH, tt.maxZoom)
			if !almostEqual(got.X, tt.want.X) ||
				!almostEqual(got.Y, tt.want.Y) ||
				!almostEqual(got.Width, tt.want.Width) ||
				!almostEqual(got.Height, tt.want.Height) ||
				!almostEqual(got.Scale, tt.want.Scale) {
				t.Errorf("FitViewport(%d, %d, %v) = %+v, want %+v",
					tt.screenW, tt.screenH, tt.maxZoom, got, tt.want)
			}
		})
	}
}
