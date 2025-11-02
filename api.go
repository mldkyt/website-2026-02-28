package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ViewHistory struct {
	YearMonth string `gorm:"primaryKey"`
	Views     int
}

type ViewExpiry struct {
	IP     string `gorm:"primaryKey"`
	Expiry time.Time
}

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

func migrate() {
	db, err := openDB()

	if err != nil {
		return
	}

	db.AutoMigrate(&ViewHistory{})
	db.AutoMigrate(&ViewExpiry{})
}

func registerApiHandlers() {
	migrate()

	http.HandleFunc("/api/views", func(w http.ResponseWriter, r *http.Request) {
		ip := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
		if ip == "" {
			ip = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
		}
		if ip == "" {
			ip = strings.TrimSpace(r.RemoteAddr)
		}

		h := sha256.New()
		h.Write([]byte(ip))
		ip = fmt.Sprintf("%x", h.Sum(nil))

		db, err := openDB()

		if err != nil {
			http.Error(w, "Unable to rw db", http.StatusInternalServerError)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		now := time.Now()

		// Delete expired rows
		db.Delete(&ViewExpiry{}, "expiry < ?", now)

		// Check cooldown expiry
		var viewExpiry ViewExpiry
		result := db.Where(&ViewExpiry{IP: ip}).First(&viewExpiry)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Update view count
			var view ViewHistory
			db.Where(ViewHistory{YearMonth: now.Format("2006-01")}).Attrs(ViewHistory{Views: 0}).FirstOrCreate(&view)
			db.Model(&view).Updates(ViewHistory{Views: view.Views + 1})
		}

		// Update cooldown
		expiry := now.Add(time.Hour)
		viewExpiry = ViewExpiry{
			IP:     ip,
			Expiry: expiry,
		}
		db.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(&viewExpiry)

		// Get views
		var views int64
		db.Model(&ViewHistory{}).Select("sum(views)").Scan(&views)

		bytes, err := json.Marshal(struct {
			Views int `json:"views"`
		}{
			Views: int(views),
		})

		if err != nil {
			http.Error(w, "What even happened in my code? Failed to serialize a number into josn :sob:", http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(bytes)
	})
}
