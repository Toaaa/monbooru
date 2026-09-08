package api

import (
	"net/http"

	"github.com/monbooru/monbooru/internal/counts"
)

// galleryListEntry is one row of GET /api/v1/galleries: a configured
// gallery the caller can target with ?gallery=<name>, plus the same
// visible-image and non-alias-tag counts the Settings page shows.
type galleryListEntry struct {
	Name   string `json:"name"`
	Images int    `json:"images"`
	Tags   int    `json:"tags"`
	Active bool   `json:"active"`
}

// listGalleries handles GET /api/v1/galleries. The set of galleries and
// the active one are derived from the configured list and the resolver
// already wired into the handler - resolver("") returns the active
// gallery - so no extra plumbing is needed. Counts are best-effort: a
// gallery whose count query fails still appears, with zero.
func (h *Handler) listGalleries(w http.ResponseWriter, r *http.Request) {
	activeName := ""
	if active, ok := h.resolver(""); ok {
		activeName = active.Name
	}
	configured := h.cfg().Galleries
	out := make([]galleryListEntry, 0, len(configured))
	for _, gc := range configured {
		g, ok := h.resolver(gc.Name)
		if !ok {
			continue
		}
		entry := galleryListEntry{Name: gc.Name, Active: gc.Name == activeName}
		entry.Images, _ = counts.VisibleCount(g.DB)
		entry.Tags, _ = counts.TagCount(g.DB)
		out = append(out, entry)
	}
	WriteJSON(w, http.StatusOK, out)
}
