package ui

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

var content embed.FS

func Handler() http.Handler {
	if _, err := os.Stat("ui/out"); os.IsNotExist(err) {
		return http.FileServer(http.FS(content))
	}

	staticFS, err := fs.Sub(content, "static/out")
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	entries, err := fs.ReadDir(staticFS, ".")
	if err != nil {
		log.Fatal("Cannot read embedded dir:", err)
	}
	for _, e := range entries {
		log.Println("Embedded file:", e.Name())
	}
	return http.FileServer(http.FS(staticFS))
}
