package web

import (
	"os"
	"path/filepath"

	"github.com/monbooru/monbooru/internal/config"
	"github.com/monbooru/monbooru/internal/logx"
	"github.com/monbooru/monbooru/internal/plugins"
)

// effectivePlugin is a config block joined with the folder that supplies its
// launch line, when one does. Installed marks the folder form, whose run
// state the operator toggles from the web and which therefore persists as
// Enabled on the block. The launch line stays off PluginConfig: it comes
// from the folder's manifest and is never written to monbooru.toml, so a
// web-writable exec line cannot exist.
type effectivePlugin struct {
	config.PluginConfig
	Launch    plugins.Launch
	Installed bool
}

// pluginsDir is where dropped plugin folders live, next to monbooru.toml.
// Absolute, because a folder-relative command becomes the launched process's
// path while its folder becomes that process's working directory: a relative
// path would then resolve against the folder instead of monbooru's cwd.
func (s *Server) pluginsDir() string { return s.configSubdir("plugins") }

// configSubdir is one of the folders monbooru keeps next to monbooru.toml,
// as an absolute path. The settings hints print these, and a `-config` given
// as a relative path would otherwise print one only the process's own working
// directory can resolve.
func (s *Server) configSubdir(name string) string {
	if s.configPath == "" {
		return ""
	}
	dir := filepath.Join(filepath.Dir(s.configPath), name)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}

// ensurePluginsDir creates the plugins folder at boot so there is somewhere
// obvious to drop one. Unlike themes it seeds nothing: an example here would
// be executable code.
func (s *Server) ensurePluginsDir() {
	dir := s.pluginsDir()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logx.Warnf("plugins: could not create %s: %v", dir, err)
	}
}

// reservePluginName refuses the folder names monbooru cannot launch: the
// companion keeps its own config section and surfaces, and a block name has
// to survive the config round trip.
func reservePluginName(name string) (string, bool) {
	if name == monloaderApp {
		return "the name belongs to the companion", true
	}
	if err := config.ValidatePluginName(name); err != nil {
		return err.Error(), true
	}
	return "", false
}

// effectivePlugins merges the configured blocks with the discovered folders
// by name: the block carries the pairing halves and the operator's flags, the
// manifest the launch line.
func (s *Server) effectivePlugins() []effectivePlugin {
	blocks := s.plugins()
	out := make([]effectivePlugin, 0, len(blocks))
	byName := make(map[string]int, len(blocks))
	for _, p := range blocks {
		byName[p.Name] = len(out)
		out = append(out, effectivePlugin{PluginConfig: p})
	}
	for _, ins := range plugins.Discover(s.pluginsDir(), reservePluginName) {
		i, ok := byName[ins.Name]
		if !ok {
			out = append(out, effectivePlugin{
				PluginConfig: config.PluginConfig{Name: ins.Name},
				Launch:       ins,
				Installed:    true,
			})
			continue
		}
		out[i].Launch, out[i].Installed = ins, true
	}
	return out
}

// effective returns the merged view of one plugin, or false.
func (s *Server) effective(name string) (effectivePlugin, bool) {
	for _, p := range s.effectivePlugins() {
		if p.Name == name {
			return p, true
		}
	}
	return effectivePlugin{}, false
}
