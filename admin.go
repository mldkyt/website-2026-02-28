package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func registerAdminHandlers() {
	adminTemplate := template.Must(template.New("admin").Delims("[[", "]]").ParseFiles("dist/admin/index.html"))

	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		err := adminTemplate.ExecuteTemplate(w, "index.html", struct{}{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
		}
	})

	adminGuestbookTemplate := template.Must(template.New("adminGuestbook").Delims("[[", "]]").ParseFiles("dist/admin/guestbook/index.html"))

	http.HandleFunc("/admin/guestbook", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		db, err := openDB()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		if query.Has("approve") {
			println("approve", query.Get("approve"))
			id, err := strconv.Atoi(query.Get("approve"))
			if err != nil {
				println("fail to parse number")
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			db.Model(&GuestbookItem{}).Where("id = ?", id).Where("approved = ?", false).Update("approved", true)
			http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
			return
		}

		if query.Has("delete") {
			println("delete", query.Get("delete"))
			id, err := strconv.Atoi(query.Get("delete"))
			if err != nil {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			db.Delete(&GuestbookItem{}, "id = ?", id)
			http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
			return
		}

		if query.Has("ban") {
			id, err := strconv.Atoi(query.Get("ban"))
			if err != nil {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			var item GuestbookItem
			result := db.First(&item, "id = ?", id)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			db.Model(&GuestbookBans{}).Create(&GuestbookBans{
				IP:      item.IP,
				FullBan: false,
			})

			db.Delete(&item)

			http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
			return
		}

		if query.Has("banfull") {
			id, err := strconv.Atoi(query.Get("banfull"))
			if err != nil {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			var item GuestbookItem
			result := db.First(&item, "id = ?", id)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			db.Model(&GuestbookBans{}).Create(&GuestbookBans{
				IP:      item.IP,
				FullBan: true,
			})

			db.Delete(&item)

			http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
			return
		}

		if query.Has("unban") {
			id, err := strconv.Atoi(query.Get("unban"))
			if err != nil {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			db.Delete(&GuestbookBans{}, "id = ?", id)

			http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
			return
		}

		// Post reply
		if r.Method == "POST" {
			r.ParseForm()
			if !r.Form.Has("id") || !r.Form.Has("reply") {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			id, err := strconv.Atoi(r.Form.Get("id"))
			if err != nil {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			var item GuestbookItem
			result := db.First(&item, "id = ?", id)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				http.Redirect(w, r, "/admin/guestbook", http.StatusTemporaryRedirect)
				return
			}

			item.Reply = r.Form.Get("reply")
			db.Model(&GuestbookItem{}).Where("id = ?", id).Update("reply", item.Reply)
		}

		// Unapproved items

		var unapproved []GuestbookItem
		db.Find(&unapproved, "approved = ?", false)

		var unapprovedItems []struct {
			Name      string
			Message   string
			ID        uint
			CreatedAt string
			IP        string
		}

		for _, v := range unapproved {
			unapprovedItems = append(unapprovedItems, struct {
				Name      string
				Message   string
				ID        uint
				CreatedAt string
				IP        string
			}{
				Name:      v.Name,
				Message:   v.Message,
				ID:        v.ID,
				CreatedAt: v.CreatedAt.Format("2006-01-02 15:04:05"),
				IP:        v.IP,
			})
		}

		// bans

		var bans []GuestbookBans
		db.Find(&bans)

		var bansItems []struct {
			ID      uint
			IP      string
			FullBan string
		} = []struct {
			ID      uint
			IP      string
			FullBan string
		}{}

		for _, v := range bans {
			var fullBanStr string
			if v.FullBan {
				fullBanStr = "Full ban"
			} else {
				fullBanStr = "Posting ban"
			}

			bansItems = append(bansItems, struct {
				ID      uint
				IP      string
				FullBan string
			}{
				ID:      v.ID,
				IP:      v.IP,
				FullBan: fullBanStr,
			})
		}

		// Replies

		var existingPosts []GuestbookItem
		db.Model(&GuestbookItem{}).Where("approved = ?", true).Order(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "created_at"}, Desc: true},
			},
		}).Limit(50).Find(&existingPosts)

		var existingPostsItems []struct {
			Name    string
			Message string
			ID      uint
			Reply   string
		}

		for _, v := range existingPosts {
			existingPostsItems = append(existingPostsItems, struct {
				Name    string
				Message string
				ID      uint
				Reply   string
			}{
				Name:    v.Name,
				Message: v.Message,
				ID:      v.ID,
				Reply:   v.Reply,
			})
		}

		_ = adminGuestbookTemplate.ExecuteTemplate(w, "index.html", struct {
			UnapprovedPosts []struct {
				Name      string
				Message   string
				ID        uint
				CreatedAt string
				IP        string
			}
			Bans []struct {
				ID      uint
				IP      string
				FullBan string
			}
			ApprovedPosts []struct {
				Name    string
				Message string
				ID      uint
				Reply   string
			}
		}{
			UnapprovedPosts: unapprovedItems,
			Bans:            bansItems,
			ApprovedPosts:   existingPostsItems,
		})
	})
}
