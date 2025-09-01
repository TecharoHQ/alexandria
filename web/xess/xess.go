// Package xess vendors a copy of Xess and makes it available at /.xess/xess.css
//
// This is intended to be used as a vendored package in other projects.
package xess

import (
	"embed"
	"net/http"

	"github.com/a-h/templ"
)

//go:generate go tool templ generate

var (
	//go:embed xess.css static
	Static embed.FS

	URL = "/static/css/xess/xess.css"
)

func init() {
	Mount(http.DefaultServeMux)
}

func Mount(mux *http.ServeMux) {
	mux.Handle("/static/css/xess/", http.StripPrefix("/static/css/xess/", http.FileServerFS(Static)))
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	templ.Handler(
		Simple("Not found: "+r.URL.Path, fourohfour(r.URL.Path)),
		templ.WithStatus(http.StatusNotFound),
	)
}
