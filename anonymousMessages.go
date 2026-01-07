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
	"time"

	"github.com/google/uuid"
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

// Cookie check struct
type CookieCheck struct {
	gorm.Model
	Code    string
	Expires time.Time
}

func checkCookie(cookie *http.Cookie) bool {

	query := db.Delete(&CookieCheck{}, "expires < ?", time.Now().Local())

	cookieCheck := &CookieCheck{}
	query = db.Where("code = ?", cookie.Value).First(&cookieCheck)
	if errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return false
	}

	return true
}

func validateCookie(r *http.Request) (string, error) {
	err := validateMCaptcha(r)
	if err != nil {
		return "", err
	}

	generatedUuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	db.Create(&CookieCheck{
		Code:    generatedUuid.String(),
		Expires: time.Now().Local().Add(time.Minute * time.Duration(30)),
	})

	return generatedUuid.String(), nil
}

func validateMCaptcha(r *http.Request) error {
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
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/api/v1/pow/siteverify", os.Getenv("MCAPTCHA_URL")), bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	rawBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var out struct {
		Valid bool `json:"valid"`
	}
	err = json.Unmarshal(rawBody, &out)
	if err != nil {
		return err
	}

	if !out.Valid {
		return fmt.Errorf("mcaptcha verification failed")
	}

	return nil
}

func registerAnonymousMessages() {
	db.AutoMigrate(&AnonymousMessage{})
	db.AutoMigrate(&AnonymousMessageBan{})
	db.AutoMigrate(&CookieCheck{})

	contactTemplate := template.Must(template.New("contact").Delims("[[", "]]").ParseFiles("dist/contact/index.html"))

	http.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		type ContactPageResponse struct {
			ShowCaptcha          bool
			SentAnonymousMessage bool
			IsError              bool
			Error                string
		}

		cookie, err := r.Cookie("access")
		if err != nil {
			cookie = &http.Cookie{
				Name:     "access",
				Value:    "",
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				HttpOnly: true,
				Path:     "/",
			}
		}

		if r.Method == "POST" {
			// Parse form

			err := r.ParseForm()
			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
					ShowCaptcha:          false,
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                "Invalid data",
				})
				return
			}

			if r.Form.Has("validateaccess") && r.Form.Get("validateaccess") == "true" {
				code, err := validateCookie(r)
				if err != nil {
					contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
						ShowCaptcha:          false,
						SentAnonymousMessage: false,
						IsError:              true,
						Error:                err.Error(),
					})
					return
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "access",
					Value:    code,
					Secure:   true,
					HttpOnly: true,
					MaxAge:   1800,
					Path:     "/",
					SameSite: http.SameSiteLaxMode,
				})

				http.Redirect(w, r, "/contact", http.StatusFound)
				return
			}

			if !checkCookie(cookie) {
				contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
					ShowCaptcha: true,
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

			// MCaptcha
			err = validateMCaptcha(r)

			if err != nil {
				contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
					ShowCaptcha:          false,
					SentAnonymousMessage: false,
					IsError:              true,
					Error:                err.Error(),
				})
			}

			// Add anonymous message to database
			var msg AnonymousMessage
			result := db.First(&msg, "ip = ?", ip)
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
					ShowCaptcha:          false,
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
				contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
					ShowCaptcha:          false,
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

			contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
				ShowCaptcha:          false,
				SentAnonymousMessage: true,
				IsError:              false,
				Error:                "",
			})
			return
		}

		if !checkCookie(cookie) {
			contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
				ShowCaptcha: true,
			})

			return
		}

		contactTemplate.ExecuteTemplate(w, "index.html", ContactPageResponse{
			ShowCaptcha:          false,
			SentAnonymousMessage: false,
			IsError:              false,
			Error:                "",
		})
	})
}
