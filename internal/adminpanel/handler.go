package adminpanel

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var distFS embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("adminpanel: failed to sub embed FS: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}
