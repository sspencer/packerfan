package render

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Art owns the loaded Raylib textures, spritesheet image, and shaders.
type Art struct {
	Font        rl.Texture2D
	Texture     rl.Texture2D
	Image       *rl.Image
	Shader      rl.Shader
	FlashShader rl.Shader
}

const flashFragmentShader = `#version 330
in vec2 fragTexCoord;
in vec4 fragColor;
out vec4 finalColor;
uniform sampler2D texture0;
void main()
{
    vec4 texColor = texture(texture0, fragTexCoord);
    if (max(max(texColor.r, texColor.g), texColor.b) > 0.05)
    {
        finalColor = vec4(1.0, 1.0, 1.0, texColor.a);
    }
    else
    {
        finalColor = vec4(0.0, 0.0, 0.0, texColor.a);
    }
}
`

// Paths of the graphical assets NewArt loads from the embedded filesystem.
const (
	fontAssetPath   = "assets/text.png"
	gameAssetPath   = "assets/game.png"
	shaderAssetPath = "assets/chroma_key.fs"
)

// RequiredAssets lists every file NewArt reads from the embedded filesystem.
var RequiredAssets = []string{fontAssetPath, gameAssetPath, shaderAssetPath}

// CheckAssets verifies that every file in RequiredAssets exists in assetsFS.
// It returns an error naming each missing file, or nil if all are present.
// Call it before NewArt so a missing asset is reported clearly instead of
// panicking mid-load.
func CheckAssets(assetsFS fs.FS) error {
	var missing []string
	for _, path := range RequiredAssets {
		info, err := fs.Stat(assetsFS, path)
		if err != nil || info.IsDir() {
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return errors.New("missing required asset file(s):\n  " + strings.Join(missing, "\n  "))
}

// NewArt loads all graphical assets from the provided embedded filesystem.
func NewArt(assetsFS fs.FS) *Art {
	fontTexture := loadTexture(assetsFS, fontAssetPath)
	gameTexture := loadTexture(assetsFS, gameAssetPath)
	gameImage := rl.LoadImageFromTexture(gameTexture)
	shader := loadShader(assetsFS, shaderAssetPath)
	flashShader := rl.LoadShaderFromMemory("", flashFragmentShader)

	return &Art{
		Font:        fontTexture,
		Texture:     gameTexture,
		Image:       gameImage,
		Shader:      shader,
		FlashShader: flashShader,
	}
}

// Unload frees all GPU/CPU resources managed by Art.
func (a *Art) Unload() {
	rl.UnloadTexture(a.Font)
	rl.UnloadTexture(a.Texture)
	rl.UnloadImage(a.Image)
	rl.UnloadShader(a.Shader)
	rl.UnloadShader(a.FlashShader)
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
