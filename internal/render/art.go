package render

import (
	"fmt"
	"io/fs"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Art owns the loaded Raylib textures, spritesheet image, and chroma-key shader.
type Art struct {
	Font    rl.Texture2D
	Texture rl.Texture2D
	Image   *rl.Image
	Shader  rl.Shader
}

// NewArt loads all graphical assets from the provided embedded filesystem.
func NewArt(assetsFS fs.FS) *Art {
	fontTexture := loadTexture(assetsFS, "assets/text.png")
	gameTexture := loadTexture(assetsFS, "assets/game.png")
	gameImage := rl.LoadImageFromTexture(gameTexture)
	shader := loadShader(assetsFS, "assets/chroma_key.fs")

	return &Art{
		Font:    fontTexture,
		Texture: gameTexture,
		Image:   gameImage,
		Shader:  shader,
	}
}

// Unload frees all GPU/CPU resources managed by Art.
func (a *Art) Unload() {
	rl.UnloadTexture(a.Font)
	rl.UnloadTexture(a.Texture)
	rl.UnloadImage(a.Image)
	rl.UnloadShader(a.Shader)
}

// IsPixelLit reports whether the pixel at (x, y) on the spritesheet image is non-black.
func (a *Art) IsPixelLit(x, y int) bool {
	c := rl.GetImageColor(*a.Image, int32(x), int32(y))
	return c.R > 0 || c.G > 0 || c.B > 0
}

func loadTexture(assetsFS fs.FS, path string) rl.Texture2D {
	data := mustReadAsset(assetsFS, path)
	img := rl.LoadImageFromMemory(".png", data, int32(len(data)))
	texture := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return texture
}

func loadShader(assetsFS fs.FS, path string) rl.Shader {
	fragmentShaderCode := mustReadAsset(assetsFS, path)
	shader := rl.LoadShaderFromMemory("", string(fragmentShaderCode))
	keyColor := []float32{0.0, 0.0, 0.0}
	threshold := []float32{0.05}

	keyColorLoc := rl.GetShaderLocation(shader, "keyColor")
	thresholdLoc := rl.GetShaderLocation(shader, "threshold")
	rl.SetShaderValue(shader, keyColorLoc, keyColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, thresholdLoc, threshold, rl.ShaderUniformFloat)
	return shader
}

func mustReadAsset(assetsFS fs.FS, path string) []byte {
	data, err := fs.ReadFile(assetsFS, path)
	if err != nil {
		panic(fmt.Sprintf("failed to read embedded asset %s: %v", path, err))
	}
	return data
}
