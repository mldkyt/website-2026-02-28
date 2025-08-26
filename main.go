package main

import (
	"net/http"
	"strings"
)

func cache(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if strings.Contains(r.URL.Path, "/_astro") || strings.HasPrefix(r.URL.Path, "/scripts") || strings.HasPrefix(r.URL.Path, "/button") || r.URL.Path == "/favicon.ico" {
			w.Header().Set("Cache-Control", "public, max-age=604800")
		}

		println(r.URL.Path)

		next(w, r)
	}
}

func main() {

	fs := http.FileServer(http.Dir("dist"))
	http.HandleFunc("/", cache(fs.ServeHTTP))

	registerAdminHandlers()

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
