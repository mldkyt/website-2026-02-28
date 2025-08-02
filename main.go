package main

import (
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("dist"))
	http.Handle("/", fs)

	registerAdminHandlers()

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
