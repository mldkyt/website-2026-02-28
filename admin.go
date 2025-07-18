package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"os"
	"text/template"
)

type AnonymousMessagesTemplate struct {
	Messages []AdminAnonymousMessage
}

type AdminAnonymousMessage struct {
	ID         string
	Content    string
	ShowButton bool
}

type ShowTemplate struct {
	ID      string
	CW      string
	Message string
}

type ReplyTemplate struct {
	ID      string
	CW      string
	Message string
	HasCW   bool
}

type DeleteTemplate struct {
	ID      string
	CW      string
	Message string
	HasCW   bool
}

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

	adminAnonymousMessagesTemplate := template.Must(template.New("admin anonymous messages").Delims("[[", "]]").ParseFiles("dist/admin/anonymousmessages/index.html"))

	http.HandleFunc("/admin/anonymousmessages", func(writer http.ResponseWriter, request *http.Request) {
		user, pass, ok := request.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			writer.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		messages := make([]AdminAnonymousMessage, 0)

		for _, e := range readAnonymousMessagesFile().Messages {
			content := e.Message
			if len(e.CW) > 0 {
				content = fmt.Sprintf("CW: %s", e.CW)
			}

			messages = append(messages, AdminAnonymousMessage{ID: e.ID, Content: content, ShowButton: len(e.CW) > 0})
		}

		err := adminAnonymousMessagesTemplate.ExecuteTemplate(writer, "index.html", AnonymousMessagesTemplate{
			Messages: messages,
		})

		if err != nil {
			http.Error(writer, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
		}
	})

	adminAnonymousMessagesShowTemplate := template.Must(template.New("admin anonymous messages show").Delims("[[", "]]").ParseFiles("dist/admin/anonymousmessages/show/index.html"))

	http.HandleFunc("/admin/anonymousmessages/show", func(writer http.ResponseWriter, request *http.Request) {
		user, pass, ok := request.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			writer.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := request.URL.Query().Get("id")

		data := readAnonymousMessagesFile().Messages

		for _, el := range data {
			if el.ID == id {
				err := adminAnonymousMessagesShowTemplate.ExecuteTemplate(writer, "index.html", ShowTemplate{
					Message: el.Message,
					ID:      el.ID,
					CW:      el.CW,
				})

				if err != nil {
					http.Error(writer, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
				}

				return
			}
		}

		http.NotFound(writer, request)
	})

	adminAnonymousMessagesReplyTemplate := template.Must(template.New("admin anonymous messages show").Delims("[[", "]]").ParseFiles("dist/admin/anonymousmessages/reply/index.html"))

	http.HandleFunc("/admin/anonymousmessages/reply", func(writer http.ResponseWriter, request *http.Request) {
		user, pass, ok := request.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			writer.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := request.URL.Query().Get("id")

		data := readAnonymousMessagesFile().Messages

		for _, el := range data {
			if el.ID == id {
				err := adminAnonymousMessagesReplyTemplate.ExecuteTemplate(writer, "index.html", ReplyTemplate{
					Message: el.Message,
					ID:      el.ID,
					CW:      el.CW,
					HasCW:   len(el.CW) > 0,
				})

				if err != nil {
					http.Error(writer, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
				}

				return
			}
		}

		http.NotFound(writer, request)
	})

	http.HandleFunc("/api/anonymous-message/reply", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Unable to parse form data", http.StatusInternalServerError)
			return
		}

		// Post ID

		id := r.PostFormValue("id")

		err = uuid.Validate(id)
		if err != nil {
			http.Error(w, "Unable to validate UUID", http.StatusInternalServerError)
			return
		}

		// Get anonymous message object

		anonymousMessages := readAnonymousMessagesFile().Messages
		anonymousMessage := AnonymousMessage{}

		for _, el := range anonymousMessages {
			if el.ID == id {
				anonymousMessage = el
			}
		}

		if len(anonymousMessage.Message) == 0 {
			http.Error(w, "Unable to find message", http.StatusInternalServerError)
			return
		}

		// Get reply

		reply := r.PostFormValue("reply")

		if len(reply) < 1 || len(reply) > 100 {
			http.Error(w, "Reply lengthe error", http.StatusInternalServerError)
			return
		}

		// Build text

		finalCw := ""
		finalText := fmt.Sprintf("> %s\n%s", anonymousMessage.Message, reply)

		if len(anonymousMessage.CW) == 0 {
			finalCw = "Anonymous Message Reply"
		} else {
			finalCw = fmt.Sprintf("Anonymous Message Reply, %s", anonymousMessage.CW)
		}

		// To JSON

		jsonData, err := json.Marshal(struct {
			CW   string `json:"cw"`
			Text string `json:"text"`
		}{
			CW:   finalCw,
			Text: finalText,
		})

		// Send request

		req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/api/notes/create", os.Getenv("FEDI_INSTANCE")), bytes.NewReader(jsonData))

		if err != nil {
			http.Error(w, "Unable to create request", http.StatusInternalServerError)
			return
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("FEDI_TOKEN")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			http.Error(w, "Unable to send request", http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {

			// Delete anonymous message

			newMessages := make([]AnonymousMessage, 0)

			for _, el := range anonymousMessages {
				if el.ID == id {
					continue
				}
				newMessages = append(newMessages, el)
			}

			err = writeAnonymousMessagesFile(AnonymousMessages{
				Messages: newMessages,
			})

			if err != nil {
				http.Error(w, "Unable to delete anonymous message", http.StatusInternalServerError)
			}

			http.Redirect(w, r, "/admin/anonymousmessages", http.StatusSeeOther)
		} else {
			http.Error(w, fmt.Sprintf("Network request failed with code %s", resp.Status), http.StatusInternalServerError)
		}

	})

	adminAnonymousMessagesDeleteTemplate := template.Must(template.New("admin anonymous message delete").Delims("[[", "]]").ParseFiles("dist/admin/anonymousmessages/delete/index.html"))

	http.HandleFunc("/admin/anonymousmessages/delete", func(writer http.ResponseWriter, request *http.Request) {
		user, pass, ok := request.BasicAuth()
		if !ok || user != os.Getenv("ADMIN_USER") || pass != os.Getenv("ADMIN_PASS") {
			writer.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := request.URL.Query().Get("id")

		data := readAnonymousMessagesFile().Messages

		for _, el := range data {
			if el.ID == id {

				if request.URL.Query().Get("confirm") == "1" {
					newData := make([]AnonymousMessage, 0)

					for _, el2 := range data {
						if el2.ID == id {
							continue
						}

						newData = append(newData, el2)
					}

					err := writeAnonymousMessagesFile(AnonymousMessages{
						Messages: newData,
					})

					if err != nil {
						http.Error(writer, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
					}

					http.Redirect(writer, request, "/admin/anonymousmessages", http.StatusSeeOther)
					return
				}

				err := adminAnonymousMessagesDeleteTemplate.ExecuteTemplate(writer, "index.html", DeleteTemplate{
					ID:      el.ID,
					CW:      el.CW,
					Message: el.Message,
					HasCW:   len(el.CW) > 0,
				})

				if err != nil {
					http.Error(writer, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
				}

				return
			}
		}

		http.NotFound(writer, request)
	})
}
