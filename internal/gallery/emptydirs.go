package gallery

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// EmptyDir is one directory under the gallery root whose subtree holds no
// file. Path is relative to the root and "/"-separated, like folder_path.
type EmptyDir struct {
	Path    string
	ModTime time.Time
}

// ScanEmptyDirs reports every directory under galleryPath holding nothing,
// ordered so a parent precedes the children it contains. The root itself is
// never reported.
//
// A directory whose name starts with a dot is neither descended into nor
// reported, and keeps its parent: .stfolder and .git are load-bearing
// exactly when they look empty. ReadDir reports a symlinked directory as a
// non-directory entry, so a linked tree counts as content and is never
// followed.
func ScanEmptyDirs(galleryPath string) ([]EmptyDir, error) {
	var out []EmptyDir
	var walk func(dir string, depth int) (bool, error)
	walk = func(dir string, depth int) (bool, error) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			// An unreadable directory keeps itself and its parent; only the
			// root's failure is the caller's to hear about.
			if depth == 0 {
				return false, err
			}
			return false, nil
		}
		empty := true
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				empty = false
				continue
			}
			childEmpty, err := walk(filepath.Join(dir, e.Name()), depth+1)
			if err != nil {
				return false, err
			}
			empty = empty && childEmpty
		}
		if !empty || depth == 0 {
			return empty, nil
		}
		rel, err := filepath.Rel(galleryPath, dir)
		if err != nil {
			return empty, nil
		}
		var mod time.Time
		if info, statErr := os.Stat(dir); statErr == nil {
			mod = info.ModTime()
		}
		out = append(out, EmptyDir{Path: filepath.ToSlash(rel), ModTime: mod})
		return empty, nil
	}
	if _, err := walk(galleryPath, 0); err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b EmptyDir) int { return cmp.Compare(a.Path, b.Path) })
	return out, nil
}

// RemoveEmptyDirs unlinks the named directories under galleryPath, deepest
// first so a nested selection collapses in one pass, and returns how many
// went. A path that escapes the root, names the root, or names anything but
// a directory is refused; one something landed in since it was listed is
// left standing, since os.Remove will not empty a directory.
func RemoveEmptyDirs(galleryPath string, paths []string) int {
	deepestFirst := slices.Clone(paths)
	slices.SortFunc(deepestFirst, func(a, b string) int {
		return strings.Count(b, "/") - strings.Count(a, "/")
	})
	removed := 0
	for _, p := range deepestFirst {
		dir, err := ResolveSubdir(galleryPath, p)
		if err != nil || filepath.Clean(dir) == filepath.Clean(galleryPath) {
			continue
		}
		// os.Remove unlinks a file just as happily as it removes a
		// directory, so the type is checked before it is called.
		if info, statErr := os.Lstat(dir); statErr != nil || !info.IsDir() {
			continue
		}
		// Only a removal this pass counts: a folder that vanished between
		// the scan and the click is not one this run took away, and the
		// handler's "N no longer empty" tail is len(paths) - removed.
		if err := os.Remove(dir); err != nil {
			continue
		}
		removed++
	}
	return removed
}
