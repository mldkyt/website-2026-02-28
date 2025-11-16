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
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GuestbookItem struct {
	gorm.Model
	Name     string
	Message  string
	Approved bool
	IP       string
	Reply    string
}

type GuestbookBans struct {
	gorm.Model
	IP      string
	FullBan bool
}

func registerGuestbookHandlers() {
	{
		db, err := openDB()
		if err != nil {
			panic(err)
		}

		db.AutoMigrate(&GuestbookItem{})
		db.AutoMigrate(&GuestbookBans{})
	}

	guestbookTemplate := template.Must(template.New("guestbook").Delims("[[", "]]").ParseFiles("dist/guestbook/index.html"))

	http.HandleFunc("/guestbook", func(w http.ResponseWriter, r *http.Request) {
		ip := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
		if ip == "" {
			ip = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
		}
		if ip == "" {
			ip = strings.TrimSpace(r.RemoteAddr)
		}

		db, err := openDB()
		if err != nil {
			http.Error(w, "Unable to open DB connection", http.StatusInternalServerError)
			return
		}

		postingBan := false
		var ban GuestbookBans
		result := db.Model(&GuestbookBans{}).Where("ip = ?", ip).First(&ban)
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if ban.FullBan {
				http.Error(w, "You are unable to view the guestbook: Full ban", http.StatusInternalServerError)
				return
			}
			postingBan = true
		}

		var users []GuestbookItem
		db.Model(&GuestbookItem{}).Where("approved = ?", true).Order(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "created_at"}, Desc: true},
			},
		}).Limit(50).Find(&users)

		var usersParsed []struct {
			Name      string
			Message   string
			CreatedAt string
			HasReply  bool
			Reply     string
		} = make([]struct {
			Name      string
			Message   string
			CreatedAt string
			HasReply  bool
			Reply     string
		}, 0)

		for _, v := range users {
			usersParsed = append(usersParsed, struct {
				Name      string
				Message   string
				CreatedAt string
				HasReply  bool
				Reply     string
			}{
				Name:      v.Name,
				Message:   v.Message,
				CreatedAt: v.CreatedAt.Format("2006-01-02 at 15:04:05"),
				HasReply:  len(v.Reply) > 0,
				Reply:     v.Reply,
			})
		}

		if r.Method == "POST" {
			if postingBan {
				http.Error(w, "You are banned from making posts", http.StatusBadRequest)
				return
			}

			var item GuestbookItem
			result := db.Where("ip = ?", ip).Where("approved = ?", false).First(&item)
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				http.Error(w, "Already filled out by you, please wait for approval before filling out again", http.StatusInternalServerError)
				return
			}

			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Unable to parse form data", http.StatusInternalServerError)
				return
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
				http.Error(w, "Unable to validate if you're a bot or not", http.StatusInternalServerError)
				return
			}

			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/api/v1/pow/siteverify", os.Getenv("MCAPTCHA_URL")), bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			if err != nil {
				http.Error(w, "Unable to validate if you're a bot or not", http.StatusInternalServerError)
				return
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				http.Error(w, "Unable to validate if you're a bot or not", http.StatusInternalServerError)
				return
			}

			defer res.Body.Close()

			rawBody, err := io.ReadAll(res.Body)
			if err != nil {
				http.Error(w, "Unable to validate if you're a bot or not", http.StatusInternalServerError)
				return
			}

			var out struct {
				Valid bool `json:"valid"`
			}
			err = json.Unmarshal(rawBody, &out)
			if err != nil {
				http.Error(w, "Unable to validate if you're a bot or not", http.StatusInternalServerError)
				return
			}

			if !out.Valid {
				http.Error(w, "Bot", http.StatusTeapot)
				return
			}

			username := r.FormValue("username")
			if len(username) < 4 || len(username) > 30 {
				http.Error(w, "Username must be between 4 and 30 letters long", http.StatusBadRequest)
				return
			}

			usernameValidator := regexp.MustCompile("^[a-zA-Z0-9]+$")

			if !usernameValidator.MatchString(username) {
				http.Error(w, "Username must be only letters and numbers", http.StatusBadRequest)
				return
			}

			message := r.FormValue("message")
			if len(message) < 4 || len(message) > 300 {
				http.Error(w, "Message must be between 4 and 300 letters long", http.StatusBadRequest)
			}

			db.Model(&GuestbookItem{}).Create(&GuestbookItem{
				Name:     username,
				Message:  message,
				Approved: false,
				IP:       ip,
				Reply:    "",
			})

			guestbookTemplate.ExecuteTemplate(w, "index.html", struct {
				SubmitSuccess bool
				Items         []struct {
					Name      string
					Message   string
					CreatedAt string
					HasReply  bool
					Reply     string
				}
			}{
				SubmitSuccess: true,
				Items:         usersParsed,
			})

			return
		}

		guestbookTemplate.ExecuteTemplate(w, "index.html", struct {
			SubmitSuccess bool
			Items         []struct {
				Name      string
				Message   string
				CreatedAt string
				HasReply  bool
				Reply     string
			}
		}{
			SubmitSuccess: false,
			Items:         usersParsed,
		})
	})
}
