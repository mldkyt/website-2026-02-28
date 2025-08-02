package main

import (
	"fmt"
	"net/http"
	"os"
	"text/template"
)

func registerAdminHandlers() {
	adminTemplate := template.Must(template.New("admin").Delims("[[", "]]").ParseFiles("dist/admin/index.html"))

	http.HandleFunc("/admin", func(writer http.ResponseWriter, request *http.Request) {
		user, pass, ok := request.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			writer.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		err := adminTemplate.ExecuteTemplate(writer, "index.html", struct{}{})
		if err != nil {
			http.Error(writer, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
		}
	})
}
