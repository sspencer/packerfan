package main

import rl "github.com/gen2brain/raylib-go/raylib"

func chromaShader(fn string) rl.Shader {
	shader := rl.LoadShader("", fn) // Empty string for vertex shader (use default)
	// Set shader uniforms
	keyColor := []float32{0.0, 0.0, 0.0} // Black (normalized RGB: 0.0 to 1.0)
	threshold := []float32{0.05}         // Tolerance for slight color variations

	keyColorLoc := rl.GetShaderLocation(shader, "keyColor")
	thresholdLoc := rl.GetShaderLocation(shader, "threshold")
	rl.SetShaderValue(shader, keyColorLoc, keyColor, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, thresholdLoc, threshold, rl.ShaderUniformFloat)
	return shader
}
