// NoFap-Bot Handler
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
	Photo    *[]PhotoSize     `json:"photo"`
}

type PhotoSize struct {
	FileId       string `json:"file_id"`
	FileUniqueId string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size"`
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

func (m *Message) CommandArguments() string {
	if !m.IsCommand() {
		return ""
	}

	entity := (*m.Entities)[0]
	if len(m.Text) == entity.Length {
		return "" // The command makes up the whole message
	}

	return m.Text[entity.Length+1:]
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

var DB *sql.DB

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/bot-handler", handleTelegramWebhook)

	db, err := sql.Open("sqlite3", "./tae.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	DB = db

	result, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS streaks (
			id integer not null primary key,
			streak integer,
			streak_start text
		);
	 	`)
	_ = result
	if err != nil {
		log.Printf("error creating table %q", err)
		return
	}

	log.Printf("Listening on PORT 8000")
	http.ListenAndServe(":8000", mux)

}

func getPictures() []string {
	return []string{"AgACAgUAAxkBAAP_YOFLIVgSfI-IbSImZE1PiSAZTacAAp2sMRt_eAlXutnBAAGWoxhtAQADAgADcwADIAQ", "AgACAgUAAxkBAAIBAAFg4Us0ZZ5J3ixGenh39pNpUZlwZAACoKwxG394CVdAqmmcK6AvoQEAAwIAA3MAAyAE", "AgACAgUAAxkBAAIBAWDhS0evf1N9p0QdCvbv7R1Q6lj4AAKhrDEbf3gJVxljkJHrntoMAQADAgADcwADIAQ"}
}

func getUser(update *Update) (int, int, string) {
	stmt, err := DB.Prepare("select id, streak, streak_start from streaks where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var id int
	var streak int
	var streakStart string
	err = stmt.QueryRow(strconv.Itoa(update.Message.User.Id)).Scan(&id, &streak, &streakStart)
	if err != nil {
		log.Printf("%q", err)
		return 0, 0, ""
	}
	return id, streak, streakStart
}

func setStreak(update *Update, newStreak int) {
	id, streak, streakStart := getUser(update)
	if id == 0 {
		tx, err := DB.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("insert into streaks(id, streak, streak_start) values(?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		t := time.Now()
		// subtract startStreak
		after := t.AddDate(0, 0, -newStreak)
		s := after.Format("2006-01-02")
		result, err := stmt.Exec(update.Message.User.Id, newStreak, s)
		if err != nil {
			log.Fatal(err)
		}
		_ = result
		tx.Commit()
	} else {
		tx, err := DB.Begin()
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := tx.Prepare("UPDATE streaks SET streak=?, streak_start=? WHERE id=?")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		t := time.Now()
		// subtract startStreak
		after := t.AddDate(0, 0, -newStreak)
		s := after.Format("2006-01-02")
		result, err := stmt.Exec(newStreak, s, update.Message.User.Id)
		if err != nil {
			log.Fatal(err)
		}
		_ = result
		tx.Commit()
	}
	_ = streak
	_ = streakStart
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
		case "start":
			{
				id, streak, streak_start := getUser(update)

				if id == 0 {
					tx, err := DB.Begin()
					if err != nil {
						log.Fatal(err)
					}
					stmt, err := tx.Prepare("insert into streaks(id, streak, streak_start) values(?, ?, ?)")
					if err != nil {
						log.Fatal(err)
					}
					defer stmt.Close()
					t := time.Now()
					s := t.Format("2006-01-02")
					result, err := stmt.Exec(update.Message.User.Id, 0, s)
					if err != nil {
						log.Fatal(err)
					}
					_ = result
					tx.Commit()
					text = "Created babyyy"
				} else {

					text = fmt.Sprintf("Your streak is this loooong: %d", streak)
				}
				_ = streak_start
			}
		case "streak":
			{

				id, streak, streak_start := getUser(update)
				if id == 0 {
					tx, err := DB.Begin()
					if err != nil {
						log.Fatal(err)
					}
					stmt, err := tx.Prepare("insert into streaks(id, streak, streak_start) values(?, ?, ?)")
					if err != nil {
						log.Fatal(err)
					}
					defer stmt.Close()
					t := time.Now().Local()
					s := t.Format("2006-01-02")
					result, err := stmt.Exec(update.Message.User.Id, 0, s)
					if err != nil {
						log.Fatal(err)
					}
					_ = result
					tx.Commit()
					text = fmt.Sprintf("Your streak is this loooong: %d", 0)
				} else {
					text = fmt.Sprintf("Your streak is this loooong: %d. Fap free since %s 🔥🔥🔥🔥🔥🔥🔥", streak, streak_start)
				}
			}

		case "horny":
			{
				// pictures := getPictures()
				// var telegramResponseBody, errTelegram = sendPhotoToTelegramChat(update.Message.Chat.Id, pictures[rand.Intn(len(pictures))])

				// if errTelegram != nil {
				// 	log.Printf("got error %s from telegram, reponse body is %s", errTelegram.Error(), telegramResponseBody)
				// } else {
				// 	log.Printf("Success with sending image to %d", update.Message.Chat.Id)
				// }
				// return
				text = "Stop. Dont do it."
			}
		case "setstreak":
			{
				processedArgCommand := update.Message.CommandArguments()
				processedArgCommand = strings.Trim(processedArgCommand, " ")
				if argLength := len(processedArgCommand); argLength == 0 {
					text = "Please provide an argument"
				} else {
					converted, err := strconv.Atoi(processedArgCommand)
					if err != nil {
						text = "Please provide a number"
					}
					if converted <= 0 {

						text = "Please provide a positive number"
					} else {

						text = "Alls well"
						setStreak(update, converted)
					}
				}
				log.Printf("%s", update.Message.CommandArguments())
			}
		case "help":
			{
				text = `
			These are my commands.
			
			/help - Get help
			/streak - Get your current streak
			/setstreak - Set streak
			/horny - Get horny pic
			/restart - Break streak :((((
			`
			}
		case "restart":
			{
				setStreak(update, 0)
				text = "It's okay bro. We got this"
			}
		default:
			{
				text = "Broooo. command??????"
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

type PhotoArguments struct {
	Photo  Photo  `json:"photo"`
	ChatId string `json:"chat_id"`
}

type Photo struct {
	FileId string `json:"file_id"`
}

func sendPhotoToTelegramChat(chatId int, fileId string) (string, error) {
	log.Printf("Sending image to chat id: %d", chatId)

	var telegramApi string = "https://api.telegram.org/bot" + os.Getenv("BOT_TOKEN") + "/sendPhoto"

	photo := Photo{FileId: fileId}
	photoArguments := PhotoArguments{Photo: photo, ChatId: strconv.Itoa(chatId)}

	data, err := json.Marshal(photoArguments)

	if err != nil {
		return "", err
	}

	transformed, err := json.MarshalIndent(photoArguments, "", "    ")
	if err != nil {
		log.Printf("error printing url object")
	}
	log.Printf("[url obj]: %s", transformed)

	client := &http.Client{}
	r, err := http.NewRequest("POST", telegramApi, strings.NewReader(string(data)))

	if err != nil {
		log.Printf("error when creating post image to the chat: %s", err.Error())
		return "", err
	}
	response, err := client.Do(r)
	if err != nil {
		log.Printf("error when posting image to the chat: %s", err.Error())
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
