package plugins

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/monbooru/monbooru/internal/logx"
)

// Manifest is the launch declaration a dropped plugin folder carries in its
// plugin.toml. The folder name is the plugin name; the manifest only says
// how to run what is inside it.
type Manifest struct {
	Command        string   `toml:"command"`
	Args           []string `toml:"args"`
	CommandWindows string   `toml:"command_windows"`
	ArgsWindows    []string `toml:"args_windows"`
}

// launchFor picks the per-OS launch line: the windows overrides on Windows
// when present, the plain keys everywhere else.
func (m Manifest) launchFor(goos string) (string, []string) {
	if goos == "windows" && m.CommandWindows != "" {
		if m.ArgsWindows != nil {
			return m.CommandWindows, m.ArgsWindows
		}
		return m.CommandWindows, m.Args
	}
	return m.Command, m.Args
}

// Discover scans dir for subfolders carrying a plugin.toml and returns the
// launch line each declares. Nothing runs on discovery; the scan only feeds
// the settings rows and the boot-start pass, which both gate on the
// operator's choice. skip refuses a folder name the caller reserves, and
// reports why for the log.
func Discover(dir string, skip func(name string) (string, bool)) []Launch {
	if dir == "" {
		return nil
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Launch
	for _, it := range items {
		if !it.IsDir() {
			continue
		}
		name := it.Name()
		folder := filepath.Join(dir, name)
		var m Manifest
		if _, err := toml.DecodeFile(filepath.Join(folder, "plugin.toml"), &m); err != nil {
			if !os.IsNotExist(err) {
				logx.Warnf("plugins: skipping %s: %v", folder, err)
			}
			continue
		}
		if skip != nil {
			if reason, refused := skip(name); refused {
				logx.Warnf("plugins: skipping %s: %s", folder, reason)
				continue
			}
		}
		command, args := m.launchFor(runtime.GOOS)
		if command == "" {
			logx.Warnf("plugins: skipping %s: plugin.toml names no command", folder)
			continue
		}
		out = append(out, Launch{
			Name:    name,
			Command: resolveCommand(folder, command),
			Args:    args,
			Dir:     folder,
		})
	}
	return out
}

// resolveCommand anchors a folder-relative launch line to its folder. A bare
// name stays a PATH lookup; an absolute path is used as written.
func resolveCommand(dir, command string) string {
	if filepath.IsAbs(command) || !strings.ContainsAny(command, `/\`) {
		return command
	}
	return filepath.Join(dir, command)
}
