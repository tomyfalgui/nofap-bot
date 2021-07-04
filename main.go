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
	Text string `json:"text"`
	Chat Chat   `json:"chat"`
	User User   `json:"from"`
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

func parseTelegramRequest(r *http.Request) (*Update, error) {
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("could not decode incoming update %s", err.Error())
		return nil, err
	}

	return &update, nil
}

func handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	var update, err = parseTelegramRequest(r)
	if err != nil {
		log.Printf("error parsing update %s", err.Error())
		return
	}

	transformed, err := json.MarshalIndent(update.Message, "", "    ")
	if err != nil {
		log.Printf("error printing user object")
	}

	log.Printf("user obj: %s", transformed)
	var telegramResponseBody, errTelegram = sendTextToTelegramChat(update.Message.Chat.Id, "hello belle")
	if errTelegram != nil {
		log.Printf("got error %s from telegram, reponse body is %s", errTelegram.Error(), telegramResponseBody)
	} else {
		log.Printf("punchline %s successfuly distributed to chat id %d", "hello", update.Message.Chat.Id)
	}

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
