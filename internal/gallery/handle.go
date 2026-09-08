// Package gallery owns the files under a gallery root: ingest, thumbnails,
// hashing, placement and the path containment every other package files
// through. Paths on disk are native; the folder_path column they resolve to
// is "/"-separated on every platform.
package gallery

import (
	"github.com/monbooru/monbooru/internal/db"
	"github.com/monbooru/monbooru/internal/tags"
)

// Handle is one open gallery: its name, where its files and thumbnails
// live, and the two services every per-image write goes through. It is the
// part of a gallery both transports describe identically, so it is defined
// here once and embedded by each - the REST layer's Gallery and the web
// layer's galleryCtx add their own transport-side fields around it rather
// than restating these five.
//
// It deliberately does not carry the relations service: internal/relations
// imports this package, so naming it here would cycle. The transports hold
// that alongside.
type Handle struct {
	Name           string
	GalleryPath    string
	ThumbnailsPath string
	DBPath         string
	DB             *db.DB
	TagSvc         *tags.Service
}
