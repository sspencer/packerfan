package audio

import (
	"encoding/binary"
	"math"
	"testing"

	"cheddarhead/assets"
)

func buildTestWAV(sampleRate, bitsPerSample, numSamples int) []byte {
	bytesPerSample := bitsPerSample / 8
	dataSize := numSamples * bytesPerSample
	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(buf[22:24], 1) // mono
	binary.LittleEndian.PutUint32(buf[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(sampleRate*bytesPerSample))
	binary.LittleEndian.PutUint16(buf[32:34], uint16(bytesPerSample))
	binary.LittleEndian.PutUint16(buf[34:36], uint16(bitsPerSample))
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	for i := range numSamples {
		if bitsPerSample == 8 {
			buf[44+i] = 128 + byte(i%16)
		} else {
			binary.LittleEndian.PutUint16(buf[44+i*2:46+i*2], uint16(i*64))
		}
	}
	return buf
}

func TestDecodeWAVPCMFormatsAndPitchResampling(t *testing.T) {
	// Classic Pac-Man WAV format: 8-bit mono 22050 Hz.
	wav8 := buildTestWAV(22050, 8, 2205)
	low, ok := decodeWAVPCM(wav8, int(math.Round(float64(defaultSampleRate)/0.94)))
	if !ok || len(low) == 0 {
		t.Fatalf("decodeWAVPCM 8-bit low pitch failed")
	}
	high, ok := decodeWAVPCM(wav8, int(math.Round(float64(defaultSampleRate)/1.06)))
	if !ok || len(high) == 0 {
		t.Fatalf("decodeWAVPCM 8-bit high pitch failed")
	}
	if len(low) <= len(high) {
		t.Errorf("expected low-pitch PCM (%d bytes) to be longer than high-pitch PCM (%d bytes)", len(low), len(high))
	}

	// Ms. Pac-Man WAV format: 16-bit mono 48000 Hz.
	wav16 := buildTestWAV(48000, 16, 4800)
	pcm16, ok := decodeWAVPCM(wav16, defaultSampleRate)
	if !ok || len(pcm16) != 4800*4 { // stereo 16-bit = 4 bytes/frame
		t.Fatalf("decodeWAVPCM 16-bit len = %d, ok = %v, want %d", len(pcm16), ok, 4800*4)
	}
}

func TestEmbeddedWAVsDecodeCleanlyIfPresent(t *testing.T) {
	for _, pacmanMode := range []bool{true, false} {
		for _, id := range AllSounds {
			data, ok := readFirstWAV(assets.FS, SoundPaths(pacmanMode, id))
			if !ok {
				continue
			}
			if _, ok := decodeWAVPCM(data, defaultSampleRate); !ok {
				t.Errorf("failed to decode embedded sound pacman=%v id=%v", pacmanMode, id)
			}
		}
	}
	for _, loop := range AllLoops {
		if data, ok := readFirstWAV(assets.FS, LoopPaths(loop)); ok {
			if _, ok := decodeWAVPCM(data, defaultSampleRate); !ok {
				t.Errorf("failed to decode embedded loop %v", loop)
			}
		}
		if data, ok := readFirstWAV(assets.FS, LoopFirstPaths(loop)); ok {
			if _, ok := decodeWAVPCM(data, defaultSampleRate); !ok {
				t.Errorf("failed to decode embedded firstloop %v", loop)
			}
		}
	}
}
