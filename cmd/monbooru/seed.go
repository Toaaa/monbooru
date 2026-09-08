package main

import (
	"os"
	"path/filepath"

	"github.com/monbooru/monbooru/internal/config"
	"github.com/monbooru/monbooru/internal/logx"
)

// inContainer reports whether the shipped image is what is running. The
// images bake the variable and nothing else sets it, which is what lets a
// config created anywhere else stop inheriting their volume layout.
func inContainer() bool {
	return os.Getenv("MONBOORU_CONTAINER") != ""
}

// hostSeed puts what monbooru creates beside the config file it was handed,
// the one folder a run outside a container can assume exists. Absolute, so
// a service unit's working directory cannot move them afterwards, and the
// gallery is created for the same reason the desktop profile creates one:
// a named but absent folder boots straight into degraded mode.
func hostSeed(configPath string) func(*config.Config) {
	dir := filepath.Dir(configPath)
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return func(cfg *config.Config) {
		cfg.Paths.DataPath = filepath.Join(dir, "data")
		cfg.Paths.ModelPath = filepath.Join(dir, "data", "models")
		gallery := filepath.Join(dir, "gallery")
		if err := os.MkdirAll(gallery, 0o755); err != nil {
			logx.Warnf("could not create a gallery folder at %s: %v", gallery, err)
		}
		cfg.Galleries[0].GalleryPath = gallery
	}
}
