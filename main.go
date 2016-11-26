package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/common/model"
	"github.com/tucnak/telebot"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"

	responseStart = "Hey, %s! I will now keep you up to date!\n" + commandHelp
	responseStop  = "Alright, %s! I won't talk to you again.\n" + commandHelp
	responseHelp  = `
I'm a drone.io bot. I can notify you about your builds.

Available commands:
` + commandStart + ` - Start listening for drone.io builds
` + commandStop + `- Stop listening for drone.io builds
`
)

var users map[int]telebot.User

func main() {
	log.Println("starting...")
	bot, err := telebot.NewBot(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatalln(err)
	}

	users = make(map[int]telebot.User)
	messages := make(chan telebot.Message, 100)
	bot.Listen(messages, 1*time.Second)

	go HTTPListenAndServe(bot)

	for message := range messages {
		switch message.Text {
		case commandStart:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
			users[message.Sender.ID] = message.Sender
			log.Printf("User %d subscribed", message.Sender.ID)
		case commandStop:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
			delete(users, message.Sender.ID)
			log.Printf("User %d unsubscribed", message.Sender.ID)
		case commandHelp:
			bot.SendMessage(message.Chat, responseHelp, nil)
		}
	}
}

func HTTPListenAndServe(bot *telebot.Bot) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var webhook notify.WebhookMessage

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&webhook)
		if err != nil {
			log.Printf("failed to decode webhook message: %v\n", err)
		}
		defer r.Body.Close()

		jsonWebhook, err := json.Marshal(webhook)
		if err != nil {
			log.Printf("failed to encode webhook for logging: %v", err)
		}
		log.Println(string(jsonWebhook))

		for _, alert := range webhook.Alerts {
			status := alert.Status
			switch status {
			case string(model.AlertFiring):
				status = "🔥 *" + strings.ToUpper(status) + "* 🔥"
			case string(model.AlertResolved):
				status = "✅ *" + strings.ToUpper(status) + "* ✅"
			}

			message := fmt.Sprintf(
				"%s\n*%s* (%s)\n%s",
				status,
				alert.Labels["alertname"],
				alert.Annotations["summary"],
				alert.Annotations["description"],
			)

			for _, user := range users {
				bot.SendMessage(user, message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
			}

		}

		w.WriteHeader(http.StatusOK)
	})

	log.Fatalln(http.ListenAndServe(":8080", nil))
}
