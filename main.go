// NoFap-Bot Handler
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

var DB *gorm.DB

type Streak struct {
	Id          uint `gorm:"primaryKey"`
	StreakStart string
}

func main() {

	// load env file
	// err := godotenv.Load(".env.local")
	// if err != nil {
	// 	log.Fatal("Error loading .env.local file")
	// }
	port := os.Getenv("PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", dbHost, dbUser, dbPassword, dbName, dbPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatal("Error connecting to database")
	}

	DB = db
	DB.AutoMigrate(&Streak{})

	mux := http.NewServeMux()
	mux.HandleFunc("/bot-handler", handleTelegramWebhook)

	if err != nil {
		log.Printf("error creating table %q", err)
		return
	}

	log.Printf("Listening on PORT 8000")
	http.ListenAndServe(":"+port, mux)

}

func getPictures() []string {
	return []string{"AgACAgUAAxkBAAP_YOFLIVgSfI-IbSImZE1PiSAZTacAAp2sMRt_eAlXutnBAAGWoxhtAQADAgADcwADIAQ", "AgACAgUAAxkBAAIBAAFg4Us0ZZ5J3ixGenh39pNpUZlwZAACoKwxG394CVdAqmmcK6AvoQEAAwIAA3MAAyAE", "AgACAgUAAxkBAAIBAWDhS0evf1N9p0QdCvbv7R1Q6lj4AAKhrDEbf3gJVxljkJHrntoMAQADAgADcwADIAQ"}
}

func getHornyStatements() []string {
	return []string{"Stop. Don't do it.", "God is watching you.", "Go get a cold shower.", "Work out Work Out. Move your butt!", "Think about your granny."}
}

func getUser(update *Update) (uint, uint, string) {
	var streak Streak
	err := DB.First(&streak, update.Message.User.Id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, 0, ""
	}

	timeFormat := "2006-01-02"

	t, _ := time.Parse(timeFormat, streak.StreakStart)
	duration := time.Since(t)
	fmt.Println(duration.Hours())
	hours := duration.Hours() / 24

	return streak.Id, uint(hours), streak.StreakStart
}

func setStreak(update *Update, newStreak string) {
	id, _, _ := getUser(update)
	if id == 0 {
		newStreak := Streak{Id: uint(update.Message.User.Id), StreakStart: newStreak}
		result := DB.Create(&newStreak)
		if result.Error != nil {
			log.Fatal(result.Error)
		}
	} else {
		var streak Streak
		DB.First(&streak, update.Message.User.Id)
		streak.StreakStart = newStreak
		DB.Save(&streak)
	}
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

	if update.Message.IsCommand() {
		var text string = ""
		switch update.Message.Command() {
		case "start":
			{
				id, streak, _ := getUser(update)

				if id == 0 {
					t := time.Now()
					s := t.Format("2006-01-02")

					newStreak := Streak{Id: uint(update.Message.User.Id), StreakStart: s}
					result := DB.Create(&newStreak)
					if result.Error != nil {
						log.Fatal(result.Error)
					}
					text = fmt.Sprintf("Your streak is this loooong: %d", streak)
				} else {

					text = fmt.Sprintf("Your streak is this loooong: %d", streak)
				}
			}
		case "streak":
			{

				id, streak, streak_start := getUser(update)
				if id == 0 {
					t := time.Now()
					s := t.Format("2006-01-02")

					newStreak := Streak{Id: uint(update.Message.User.Id), StreakStart: s}
					result := DB.Create(&newStreak)
					if result.Error != nil {
						log.Fatal(result.Error)
					}
					text = fmt.Sprintf("Your streak is this loooong: %d", 0)
				} else {
					text = fmt.Sprintf("%d day streak \n\nFap free since %s 🔥🔥🔥🔥🔥🔥🔥", streak, streak_start)
				}
			}

		case "horny":
			{
				statements := getHornyStatements()
				text = statements[rand.Intn(len(statements))]
			}
		case "setstreak":
			{
				processedArgCommand := update.Message.CommandArguments()
				processedArgCommand = strings.Trim(processedArgCommand, " ")
				if argLength := len(processedArgCommand); argLength == 0 {
					text = "Please provide a date. Try /setstreak YYYY-MM-DD"
				} else {
					layout := "2006-01-02"
					t, err := time.Parse(layout, processedArgCommand)
					if err != nil {
						var sb strings.Builder
						sb.WriteString("Please provide a date in the proper format.\n\n")
						sb.WriteString("Try /setstreak YYYY-MM-DD")
						text = sb.String()
					} else {
						t2, _ := time.Parse("2006-01-02", t.Format("2006-01-02"))
						duration := time.Since(t2)
						if duration.Hours() < 0 {
							text = "That date is in the future. Don't fool yourself."
						} else {

							text = "Streak updated!!!"
							setStreak(update, t.Format("2006-01-02"))
						}

					}
				}
			}
		case "help":
			{
				var sb strings.Builder

				sb.WriteString("These are my commands.\n")
				sb.WriteString("\n")
				sb.WriteString("/help - List commands\n")
				sb.WriteString("/streak - Get your current streak\n")
				sb.WriteString("/setstreak - Set streak YYYY-MM-DD\n")
				sb.WriteString("/horny - Get horny pic\n")
				sb.WriteString("/restart - Break streak :((((\n")

				text = sb.String()
			}
		case "restart":
			{
				setStreak(update, time.Now().Format("2006-01-02"))
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
	} else {
		var telegramResponseBody, errTelegram = sendTextToTelegramChat(update.Message.Chat.Id, "Check out our commands by typing /help")

		if errTelegram != nil {
			log.Printf("got error %s from telegram, reponse body is %s", errTelegram.Error(), telegramResponseBody)
		} else {
			log.Printf("Success with sending %s to %d", "Check out our commands by typing /help", update.Message.Chat.Id)
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
