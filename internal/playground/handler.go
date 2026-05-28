package playground

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns a file server for all embedded assets (JS, CSS, fonts, etc.)
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("playground: failed to sub embed FS: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}

// ServeIndex serves index.html — used for any SPA route that needs the shell.
func ServeIndex(w http.ResponseWriter, r *http.Request) {
	f, err := distFS.Open("dist/index.html")
	if err != nil {
		http.Error(w, "playground not built", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "playground not built", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", stat.ModTime(), f.(io.ReadSeeker))
}
