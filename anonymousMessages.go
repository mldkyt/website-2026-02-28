package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"

	"gorm.io/gorm"
)

type AnonymousMessage struct {
	gorm.Model
	Message string
	IP      string
}

type AnonymousMessageBan struct {
	gorm.Model
	IP string
}

func registerAnonymousMessages() {
	{
		db, err := openDB()
		if err != nil {
			panic(err)
		}

		db.AutoMigrate(&AnonymousMessage{})
		db.AutoMigrate(&AnonymousMessageBan{})
	}

	contactTemplate := template.Must(template.New("contact").Delims("[[", "]]").ParseFiles("dist/contact/index.html"))

	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// Parse form

			err := r.ParseForm()
			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Invalid data",
				})
				return
			}

			// Get IP

			ip := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
			if ip == "" {
				ip = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
			}
			if ip == "" {
				ip = strings.TrimSpace(r.RemoteAddr)
			}

			// Validate mCaptcha
			payload, err := json.Marshal(struct {
				Token  string `json:"token"`
				Key    string `json:"key"`
				Secret string `json:"secret"`
			}{
				Token:  r.FormValue("mcaptcha__token"),
				Key:    os.Getenv("MCAPTCHA_SITEKEY"),
				Secret: os.Getenv("MCAPTCHA_SECRET"),
			})

			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Captcha currently unavailable",
				})
				return
			}

			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/api/v1/pow/siteverify", os.Getenv("MCAPTCHA_URL")), bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Captcha currently unavailable",
				})
				return
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Captcha currently unavailable",
				})
				return
			}

			defer res.Body.Close()

			rawBody, err := io.ReadAll(res.Body)
			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Captcha currently unavailable",
				})
				return
			}

			var out struct {
				Valid bool `json:"valid"`
			}
			err = json.Unmarshal(rawBody, &out)
			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Captcha currently unavailable",
				})
				return
			}

			if !out.Valid {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Captcha invalid, try again",
				})
				return
			}

			// Check anonymous message
			db, err := openDB()

			var msg AnonymousMessage
			result := db.First(&msg, "ip = ?", ip)
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "You already sent one message that hasn't been answered yet, please try again after it gets answered",
				})
				return
			}

			// Check ban
			var ban AnonymousMessageBan
			result = db.First(&ban, "ip = ?", ip)
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				contactTemplate.ExecuteTemplate(w, "index.html", struct {
					SentAnonymousMessage bool
					IsError              bool
					Error                string
				}{
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "You are banned from posting anonymous messages",
				})
				return
			}

			// Insert the anonymous message

			db.Create(&AnonymousMessage{
				Message: r.FormValue("message"),
				IP:      ip,
			})

			contactTemplate.ExecuteTemplate(w, "index.html", struct {
				SentAnonymousMessage bool
				IsError              bool
				Error                string
			}{
				SentAnonymousMessage: true,
				IsError:              false,
				Error:                "",
			})
			return
		}

		contactTemplate.ExecuteTemplate(w, "index.html", struct {
			SentAnonymousMessage bool
			IsError              bool
			Error                string
		}{
			SentAnonymousMessage: false,
			IsError:              false,
			Error:                "",
		})
	})
}
