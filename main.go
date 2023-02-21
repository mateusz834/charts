package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"

	"github.com/mateusz834/charts/templates"
)

//go:embed assets
var assets embed.FS

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	http.Handle("/assets/", http.FileServer(http.FS(assets)))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		if err := templates.Index(w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	return http.ListenAndServe("localhost:8888", nil)
}
