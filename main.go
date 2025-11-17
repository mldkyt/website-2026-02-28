package main

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openDB() (db *gorm.DB, err error) {
	dbMode := os.Getenv("DB_BACKEND")

	switch dbMode {
	case "sqlite":
		{
			dbPath := os.Getenv("DB_SQLITE_PATH")

			return gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		}
	case "postgres":
		{
			dbHost := os.Getenv("DB_POSTGRES_HOST")
			dbUser := os.Getenv("DB_POSTGRES_USER")
			dbPassword := os.Getenv("DB_POSTGRES_PASSWORD")
			dbName := os.Getenv("DB_POSTGRES_NAME")
			dbPort := os.Getenv("DB_POSTGRES_PORT")
			if strings.TrimSpace(dbPort) == "" {
				dbPort = "5432"
			}

			dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", dbHost, dbUser, dbPassword, dbName, dbPort)
			return gorm.Open(postgres.Open(dsn), &gorm.Config{})
		}

	}

	return nil, fmt.Errorf("error: unsupported DB mode: %s", dbMode)
}

func HandleWebsite(w http.ResponseWriter, r *http.Request) {
	file_path := path.Clean(r.URL.Path)

	if strings.Contains(file_path, "..") {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/") && r.URL.Path != "/" {
		url_without_end, _ := strings.CutSuffix(r.URL.Path, "/")

		if r.URL.RawQuery != "" {
			url_without_end = url_without_end + "?" + r.URL.RawQuery
		}

		http.Redirect(w, r, url_without_end, http.StatusPermanentRedirect)
	}

	if strings.Contains(r.URL.Path, "/_astro") || strings.HasPrefix(r.URL.Path, "/scripts") || strings.HasPrefix(r.URL.Path, "/button") || r.URL.Path == "/favicon.ico" {
		w.Header().Set("Cache-Control", "public, max-age=604800")
	}

	dist_folder_root, err := filepath.Abs("dist")
	if err != nil {
		http.Error(w, "Error trying to find the dist folder lmao", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")

	file_path = filepath.Join("dist", r.URL.Path, "index.html")
	_, err = os.Stat(file_path)
	if err != nil {
		file_path = filepath.Join("dist", r.URL.Path)
		_, err = os.Stat(file_path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	target_file_root, err := filepath.Abs(file_path)
	if err != nil {
		http.Error(w, "Error trying to find the target file", http.StatusInternalServerError)
		return
	}

	target_file_root, err = filepath.EvalSymlinks(target_file_root)
	if err != nil {
		http.Error(w, "Error trying to find the target file", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(target_file_root, dist_folder_root) {
		http.NotFound(w, r)
		return
	}

	if ctype := mime.TypeByExtension(filepath.Ext(file_path)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}

	http.ServeFile(w, r, file_path)
}

func main() {
	registerGuestbookHandlers()
	registerAdminHandlers()
	registerApiHandlers()
	registerAnonymousMessages()

	http.HandleFunc("/", http.HandlerFunc(HandleWebsite))

	println("Starting website")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
