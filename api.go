package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
			http.Error(w, fmt.Sprint("Unable to rw db: ", err), http.StatusInternalServerError)
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
