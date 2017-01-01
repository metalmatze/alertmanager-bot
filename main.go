package main

import (
	"fmt"
	"log"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/hako/durafmt"
	"github.com/joho/godotenv"
	"github.com/tucnak/telebot"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"
	commandUsers = "/users"

	commandStatus     = "/status"
	commandAlerts     = "/alerts"
	commandSilences   = "/silences"
	commandSilenceAdd = "/silence_add"
	commandSilence    = "/silence"
	commandSilenceDel = "/silence_del"

	responseStart = "Hey, %s! I will now keep you up to date!\n" + commandHelp
	responseStop  = "Alright, %s! I won't talk to you again.\n" + commandHelp
	responseHelp  = `
I'm a Prometheus AlertManager bot for Telegram. I will notify you about alerts.
You can also ask me about my ` + commandStatus + `, ` + commandAlerts + ` & ` + commandSilences + `

Available commands:
` + commandStart + ` - Subscribe for alerts.
` + commandStop + ` - Unsubscribe for alerts.
` + commandStatus + ` - Print the current status.
` + commandAlerts + ` - List all alerts.
` + commandSilences + ` - List all silences.
`
)

var (
	BuildTime string
	Commit    string
)

// Config knows all configurations from ENV
type Config struct {
	AlertmanagerURL string `arg:"env:ALERTMANAGER_URL"`
	TelegramToken   string `arg:"env:TELEGRAM_TOKEN"`
	TelegramAdmin   int    `arg:"env:TELEGRAM_ADMIN"`
	Store           string `arg:"env:STORE"`
	ListenAddr      string `arg:"env:LISTEN_ADDR"`
}

func main() {
	log.Println("starting alertmanager-telegram")
	log.Printf("BuildTime: %s, Commit: %s\n", BuildTime, Commit)

	if err := godotenv.Load(); err != nil {
		log.Println(err)
	}

	var c Config
	arg.MustParse(&c)

	bot, err := telebot.NewBot(c.TelegramToken)
	if err != nil {
		log.Fatalln(err)
	}

	users, err := NewUserStore(c.Store)
	if err != nil {
		log.Fatalln(err)
	}

	messages := make(chan telebot.Message, 100)
	bot.Listen(messages, 1*time.Second)

	go WebhookListen(c.ListenAddr, bot, users)

	for message := range messages {
		if message.Sender.ID != c.TelegramAdmin {
			log.Printf("dropped message from unallowed sender: %s(%d)", message.Sender.Username, message.Sender.ID)
			continue
		}

		switch message.Text {
		case commandStart:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
			users.Add(message.Sender)
			log.Printf("User %s(%d) subscribed", message.Sender.Username, message.Sender.ID)
		case commandStop:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
			users.Remove(message.Sender)
			log.Printf("User %s(%d) unsubscribed", message.Sender.Username, message.Sender.ID)
		case commandHelp:
			bot.SendMessage(message.Chat, responseHelp, nil)
		case commandUsers:
			bot.SendMessage(message.Chat, fmt.Sprintf("Currently %d users are subscribed.", users.Len()), nil)
		case commandStatus:
			s, err := status(c)
			if err != nil {
				bot.SendMessage(message.Chat, fmt.Sprintf("failed to get status... %v", err), nil)
			}

			uptime := durafmt.Parse(time.Since(s.Data.Uptime))

			bot.SendMessage(
				message.Chat,
				fmt.Sprintf("Version: %s\nUptime: %s", s.Data.VersionInfo.Version, uptime),
				nil,
			)
		case commandAlerts:
			alerts, err := listAlerts(c)
			if err != nil {
				bot.SendMessage(message.Chat, fmt.Sprintf("failed to list alerts... %v", err), nil)
			}

			if len(alerts) == 0 {
				bot.SendMessage(message.Chat, "No alerts right now! 🎉", nil)
			}

			var out string
			for _, a := range alerts {
				out = out + AlertMessage(a) + "\n"
			}

			bot.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		case commandSilences:
			silences, err := listSilences(c)
			if err != nil {
				bot.SendMessage(message.Chat, fmt.Sprintf("failed to list silences... %v", err), nil)
			}

			if len(silences) == 0 {
				bot.SendMessage(message.Chat, "No silences right now.", nil)
			}

			var out string
			for _, silence := range silences {
				out = out + SilenceMessage(silence) + "\n"
			}

			bot.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		default:
			bot.SendMessage(message.Chat, "Sorry, I don't understand...", nil)
		}
	}
}
