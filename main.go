// NoFap-Bot Handler
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

// Update is a Telegram object that the handler receives every time a user interacts with our bot
type Update struct {
	UpdateId int     `json:"update_id"`
	Message  Message `json:"message"`
}

// Message is a telegram object that can be found in an update.
type Message struct {
	Text     string           `json:"text"`
	Chat     Chat             `json:"chat"`
	User     User             `json:"from"`
	Entities *[]MessageEntity `json:"entities"`
}

// IsCommand checks if a message is a command
func (m *Message) IsCommand() bool {
	if m.Entities == nil || len(*m.Entities) == 0 {
		return false
	}

	entity := (*m.Entities)[0]
	return entity.Offset == 0 && entity.IsCommand()
}

// Command reads the command sent by the user
func (m *Message) Command() string {
	if !m.IsCommand() {
		return ""
	}

	entity := (*m.Entities)[0]
	return m.Text[1:entity.Length]

}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

func (e MessageEntity) IsCommand() bool {
	return e.Type == "bot_command"
}

// User is a telegram object that represents a user
type User struct {
	Id        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// A telegram Chat indicates the conversation to which the message belongs
type Chat struct {
	Id int `json:"id"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/bot-handler", handleTelegramWebhook)
	log.Printf("Listening on PORT 8000")
	http.ListenAndServe(":8000", mux)

}

func handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	var update, err = parseTelegramRequest(r)
	if err != nil {
		log.Printf("error parsing update %s", err.Error())
		return
	}

	if update.Message.Text == "" {
		log.Print("No message")
		return
	}

	transformed, err := json.MarshalIndent(update.Message, "", "    ")
	if err != nil {
		log.Printf("error printing user object")
	}

	log.Printf("[user obj]: %s", transformed)
	if update.Message.IsCommand() {
		var text string = ""
		switch update.Message.Command() {
		case "tite":
			{
				text = "maliit tite mo whahaha"
			}
		}
		var telegramResponseBody, errTelegram = sendTextToTelegramChat(update.Message.Chat.Id, text)

		if errTelegram != nil {
			log.Printf("got error %s from telegram, reponse body is %s", errTelegram.Error(), telegramResponseBody)
		} else {
			log.Printf("Success with sending %s to %d", text, update.Message.Chat.Id)
		}
	}

}

func parseTelegramRequest(r *http.Request) (*Update, error) {
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("could not decode incoming update %s", err.Error())
		return nil, err
	}

	return &update, nil
}

func sendTextToTelegramChat(chatId int, text string) (string, error) {
	log.Printf("Sending %s to chat id: %d", text, chatId)
	var telegramApi string = "https://api.telegram.org/bot" + os.Getenv("BOT_TOKEN") + "/sendMessage"
	response, err := http.PostForm(
		telegramApi,
		url.Values{
			"chat_id": {strconv.Itoa(chatId)},
			"text":    {text},
		},
	)

	if err != nil {
		log.Printf("error when posting text to the chat: %s", err.Error())
		return "", err
	}

	defer response.Body.Close()

	var bodyBytes, errRead = io.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("error in parsing telegram answer %s", errRead.Error())
		return "", err
	}

	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)

	return bodyString, nil
}
