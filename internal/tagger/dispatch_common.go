package tagger

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/monbooru/monbooru/internal/logx"
)

// dispatchSchemaVersion is the only schema version the dispatch loader
// accepts. Documents on a different version are skipped so a future
// schema change can roll out without breaking older binaries that still
// ship the old embedded defaults.
const dispatchSchemaVersion = 1

type dispatchDoc struct {
	Version int             `json:"version"`
	Rules   []DispatchEntry `json:"rules"`
}

// DispatchEntry is one rule of a dispatch document: route the model's
// Source label into Category (empty = drop the label), optionally
// renaming it to Name. The same shape serves the embedded defaults,
// the on-disk overlay, and the settings mappings editor.
type DispatchEntry struct {
	Source   string `json:"source"`
	Category string `json:"category"`
	Name     string `json:"name,omitempty"`
}

// DispatchTargetCategories returns the distinct destination category
// names the tagger's dispatch table (embedded default + on-disk
// overlay) routes labels into. The Configure dialog unions these with
// the profile's natively emitted categories so an operator can tune or
// disable a category the model only reaches through dispatch - e.g.
// wd-swinv2 routes a slice of its general labels into medium / meta /
// year. Empty-category entries (drops) are skipped; order is
// deterministic (embedded rules first, then overlay additions).
func DispatchTargetCategories(modelPath, taggerName string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(entries []DispatchEntry) {
		for _, e := range entries {
			if e.Category == "" || seen[e.Category] {
				continue
			}
			seen[e.Category] = true
			out = append(out, e.Category)
		}
	}
	add(EmbeddedDispatchRules(taggerName))
	add(OverlayDispatchRules(modelPath, taggerName))
	return out
}

// OverlayRuleCount reports how many dispatch rules the tagger's on-disk
// overlay carries. Zero means the tagger runs on the embedded defaults
// alone; the settings table uses that as its "differs from stock"
// signal and the row summary shows the count.
func OverlayRuleCount(modelPath, taggerName string) int {
	return len(OverlayDispatchRules(modelPath, taggerName))
}

// EmbeddedDispatchRules returns the shipped default rules for one
// tagger, in file order. Empty for taggers without an embedded table.
func EmbeddedDispatchRules(taggerName string) []DispatchEntry {
	data, err := defaultDispatchFS.ReadFile("dispatch_default/" + taggerName + ".json")
	if err != nil {
		return nil
	}
	return parseDispatchDoc(data, "embedded "+taggerName)
}

// OverlayDispatchRules returns the rules of the tagger's on-disk
// overlay, in file order. Empty when no overlay exists.
func OverlayDispatchRules(modelPath, taggerName string) []DispatchEntry {
	p := filepath.Join(modelPath, taggerName, "dispatch.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	return parseDispatchDoc(data, p)
}

func parseDispatchDoc(data []byte, source string) []DispatchEntry {
	var doc dispatchDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		logx.Warnf("tagger: %s dispatch parse failed: %v", source, err)
		return nil
	}
	if doc.Version != dispatchSchemaVersion {
		logx.Warnf("tagger: %s dispatch schema version %d unsupported (want %d)", source, doc.Version, dispatchSchemaVersion)
		return nil
	}
	return doc.Rules
}
