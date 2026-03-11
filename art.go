package main

import (
	"embed"
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var assetsFS embed.FS

type Art struct {
	font    rl.Texture2D
	texture rl.Texture2D
	image   *rl.Image
	shader  rl.Shader
}

func NewArt() *Art {
	fontTexture := loadTexture("assets/text.png")
	gameTexture := loadTexture("assets/game.png")
	gameImage := rl.LoadImageFromTexture(gameTexture)
	shader := loadShader("assets/chroma_key.fs")

	return &Art{
		font:    fontTexture,
		texture: gameTexture,
		image:   gameImage,
		shader:  shader,
	}
}

func (a *Art) unload() {
	rl.UnloadTexture(a.font)
	rl.UnloadTexture(a.texture)
	rl.UnloadImage(a.image)
	rl.UnloadShader(a.shader)
}

func loadTexture(path string) rl.Texture2D {
	data := mustReadAsset(path)
	img := rl.LoadImageFromMemory(".png", data, int32(len(data)))
	texture := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return texture
}

func loadShader(path string) rl.Shader {
	fragmentShaderCode := mustReadAsset(path)
	shader := rl.LoadShaderFromMemory("", string(fragmentShaderCode)) // Empty string for vertex shader (use default)
	// Set shader uniforms
	keyColor := []float32{0.0, 0.0, 0.0} // Black (normalized RGB: 0.0 to 1.0)
	threshold := []float32{0.05}         // Tolerance for slight color variations

	keyColorLoc := rl.GetShaderLocation(shader, "keyColor")
	thresholdLoc := rl.GetShaderLocation(shader, "threshold")
	rl.SetShaderValue(shader, keyColorLoc, keyColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, thresholdLoc, threshold, rl.ShaderUniformFloat)
	return shader
}

func mustReadAsset(path string) []byte {
	data, err := assetsFS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to read embedded asset %s: %v", path, err))
	}
	return data
}
