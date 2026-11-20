package assets

import (
	"embed"
	"io"
	"io/fs"
	"strings"
	"time"
)

//go:embed *
var rawFS embed.FS

// FS exposes the embedded assets directory under the "assets/" path prefix
// (e.g. "assets/game.png", "assets/text.png", "assets/chroma_key.fs").
var FS fs.FS = prefixFS{inner: rawFS}

const (
	// RealGameAssetPath is the optional untracked original spritesheet path.
	RealGameAssetPath = "assets/game-real.png"
	// GameAssetFallbackPath is the tracked placeholder spritesheet path.
	GameAssetFallbackPath = "assets/game.png"
)

// GameAssetPath returns the game spritesheet path in assetsFS:
// assets/game-real.png if it exists as a regular file, otherwise assets/game.png.
func GameAssetPath(assetsFS fs.FS) string {
	if IsAssetFile(assetsFS, RealGameAssetPath) {
		return RealGameAssetPath
	}
	return GameAssetFallbackPath
}

// IsAssetFile reports whether path exists as a regular file in assetsFS.
func IsAssetFile(assetsFS fs.FS, path string) bool {
	if assetsFS == nil {
		return false
	}
	info, err := fs.Stat(assetsFS, path)
	return err == nil && !info.IsDir()
}

type prefixFS struct {
	inner embed.FS
}

func (p prefixFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if name == "." {
		return &rootDir{}, nil
	}
	if name == "assets" {
		f, err := p.inner.Open(".")
		if err != nil {
			return nil, err
		}
		return renamedDir{ReadDirFile: f.(fs.ReadDirFile), name: "assets"}, nil
	}
	if rest, ok := strings.CutPrefix(name, "assets/"); ok {
		return p.inner.Open(rest)
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func (p prefixFS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrInvalid}
	}
	if rest, ok := strings.CutPrefix(name, "assets/"); ok {
		return p.inner.ReadFile(rest)
	}
	return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
}

func (p prefixFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	if name == "." {
		return []fs.DirEntry{assetsDirEntry{}}, nil
	}
	if name == "assets" {
		return p.inner.ReadDir(".")
	}
	if rest, ok := strings.CutPrefix(name, "assets/"); ok {
		return p.inner.ReadDir(rest)
	}
	return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
}

type renamedDir struct {
	fs.ReadDirFile
	name string
}

func (d renamedDir) Stat() (fs.FileInfo, error) { return dirInfo(d.name), nil }

type rootDir struct {
	read bool
}

func (d *rootDir) Stat() (fs.FileInfo, error) { return dirInfo("."), nil }
func (d *rootDir) Read([]byte) (int, error)   { return 0, io.EOF }
func (d *rootDir) Close() error               { return nil }
func (d *rootDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.read {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	d.read = true
	return []fs.DirEntry{assetsDirEntry{}}, nil
}

type assetsDirEntry struct{}

func (assetsDirEntry) Name() string               { return "assets" }
func (assetsDirEntry) IsDir() bool                { return true }
func (assetsDirEntry) Type() fs.FileMode          { return fs.ModeDir }
func (assetsDirEntry) Info() (fs.FileInfo, error) { return dirInfo("assets"), nil }

type dirInfo string

func (d dirInfo) Name() string       { return string(d) }
func (d dirInfo) Size() int64        { return 0 }
func (d dirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (d dirInfo) ModTime() time.Time { return time.Time{} }
func (d dirInfo) IsDir() bool        { return true }
func (d dirInfo) Sys() any           { return nil }
