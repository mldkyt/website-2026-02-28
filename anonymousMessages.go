package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"os"
	"text/template"
)

type AnonymousMessage struct {
	ID      string `json:"id"`
	CW      string `json:"cw"`
	Message string `json:"message"`
	IP      string `json:"ip"`
}

type AnonymousMessages struct {
	Messages []AnonymousMessage `json:"messages"`
}

func readAnonymousMessagesFile() AnonymousMessages {
	bytesData, err := os.ReadFile("data/anonymousmessages.json")
	if err != nil {
		return AnonymousMessages{}
	}

	var data AnonymousMessages
	err = json.Unmarshal(bytesData, &data)
	if err != nil {
		return AnonymousMessages{}
	}

	return data
}

func writeAnonymousMessagesFile(data AnonymousMessages) error {
	bytesData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = os.MkdirAll("data", 0755)
	if err != nil {
		return err
	}

	openFile, err := os.OpenFile("data/anonymousmessages.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = openFile.Write(bytesData)
	if err != nil {
		return err
	}

	return openFile.Close()
}

func registerAnonymousMessagesHandlers() {
	http.HandleFunc("/api/anonymous-message", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Redirect(w, r, "/anonymous-message?err=1", http.StatusSeeOther)
			return
		}

		cw := r.PostForm.Get("cw")
		message := r.PostForm.Get("message")
		ip := r.Header.Get("X-Forwarded-For")

		if len(message) < 1 || len(message) > 100 || len(cw) > 100 {
			log.Printf("/api/anonymous-message error: Values are invalid")
			http.Redirect(w, r, "/anonymous-message?err=1", http.StatusSeeOther)
			return
		}

		generatedUuid, err := uuid.NewRandom()
		if err != nil {
			http.Redirect(w, r, "/anonymous-message?err=1", http.StatusSeeOther)
			return
		}

		data := readAnonymousMessagesFile()
		data.Messages = append(data.Messages, AnonymousMessage{
			ID:      generatedUuid.String(),
			CW:      cw,
			Message: message,
			IP:      ip,
		})
		err = writeAnonymousMessagesFile(data)

		if err != nil {
			log.Println("/api/anonymous-message error: ", err)
			http.Redirect(w, r, "/anonymous-message?err=1", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/anonymous-message?ok=1", http.StatusSeeOther)
	})

	anonymousMessageTemplate := template.Must(template.New("anonymous message").Delims("[[", "]]").ParseFiles("dist/anonymous-message/index.html"))

	http.HandleFunc("/anonymous-message", func(writer http.ResponseWriter, request *http.Request) {
		queryErr := request.URL.Query().Get("err")
		queryOk := request.URL.Query().Get("ok")

		err := anonymousMessageTemplate.ExecuteTemplate(writer, "index.html", struct {
			Error bool
			OK    bool
		}{
			Error: len(queryErr) > 0,
			OK:    len(queryOk) > 0,
		})

		if err != nil {
			http.Error(writer, fmt.Sprintf("Error: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	})
}
