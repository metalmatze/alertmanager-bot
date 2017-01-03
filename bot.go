package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hako/durafmt"
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
I'm a Prometheus AlertManager telegram for Telegram. I will notify you about alerts.
You can also ask me about my ` + commandStatus + `, ` + commandAlerts + ` & ` + commandSilences + `

Available commands:
` + commandStart + ` - Subscribe for alerts.
` + commandStop + ` - Unsubscribe for alerts.
` + commandStatus + ` - Print the current status.
` + commandAlerts + ` - List all alerts.
` + commandSilences + ` - List all silences.
`
)

// Bot runs the alertmanager telegram
type Bot struct {
	Config    Config
	UserStore *UserStore

	telegram *telebot.Bot
}

// NewBot creates a Bot with the UserStore and telegram telegram
func NewBot(c Config) (*Bot, error) {
	users, err := NewUserStore(c.Store)
	if err != nil {
		return nil, err
	}

	bot, err := telebot.NewBot(c.TelegramToken)
	if err != nil {
		return nil, err
	}

	return &Bot{
		telegram:  bot,
		Config:    c,
		UserStore: users,
	}, nil
}

// RunWebhook starts a http server and listens for messages to send to the users
func (b *Bot) RunWebhook() {
	messages := make(chan string, 100)

	http.HandleFunc("/", HandleWebhook(messages))

	go b.sendWebhook(messages)

	addr := ":8080"
	if b.Config.ListenAddr != "" {
		addr = b.Config.ListenAddr
	}

	log.Fatalln(http.ListenAndServe(addr, nil))
}

// sendWebhook sends messages received via webhook to all subscribed users
func (b *Bot) sendWebhook(messages <-chan string) {
	for m := range messages {
		for _, user := range b.UserStore.List() {
			b.telegram.SendMessage(user, m, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		}
	}
}

// Run the telegram and listen to messages send to the telegram
func (b *Bot) Run() {
	messages := make(chan telebot.Message, 100)
	b.telegram.Listen(messages, time.Second)

	for message := range messages {
		if message.Sender.ID != b.Config.TelegramAdmin {
			log.Printf("dropped message from unallowed sender: %s(%d)", message.Sender.Username, message.Sender.ID)
			continue
		}

		b.telegram.SendChatAction(message.Chat, telebot.Typing)

		switch message.Text {
		case commandStart:
			b.handleStart(message)
		case commandStop:
			b.handleStop(message)
		case commandHelp:
			b.handleHelp(message)
		case commandUsers:
			b.handleUsers(message)
		case commandStatus:
			b.handleStatus(message)
		case commandAlerts:
			b.handleAlerts(message)
		case commandSilences:
			b.handleSilences(message)
		default:
			b.telegram.SendMessage(
				message.Chat,
				"Sorry, I don't understand...",
				nil,
			)
		}
	}
}

func (b *Bot) handleStart(message telebot.Message) {
	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
	b.UserStore.Add(message.Sender)
	log.Printf("User %s(%d) subscribed", message.Sender.Username, message.Sender.ID)
}

func (b *Bot) handleStop(message telebot.Message) {
	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
	b.UserStore.Remove(message.Sender)
	log.Printf("User %s(%d) unsubscribed", message.Sender.Username, message.Sender.ID)
}

func (b *Bot) handleHelp(message telebot.Message) {
	b.telegram.SendMessage(message.Chat, responseHelp, nil)
}

func (b *Bot) handleUsers(message telebot.Message) {
	b.telegram.SendMessage(message.Chat, fmt.Sprintf(
		"Currently %d users are subscribed.",
		b.UserStore.Len()),
		nil,
	)
}

func (b *Bot) handleStatus(message telebot.Message) {
	s, err := status(b.Config.AlertmanagerURL)
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to get status... %v", err), nil)
		return
	}

	uptime := durafmt.Parse(time.Since(s.Data.Uptime))
	uptimeBot := durafmt.Parse(time.Since(StartTime))

	b.telegram.SendMessage(
		message.Chat,
		fmt.Sprintf(
			"*AlertManager*\nVersion: %s\nUptime: %s\n*AlertManager Bot*\nVersion: %s\nUptime: %s",
			s.Data.VersionInfo.Version,
			uptime,
			Commit,
			uptimeBot,
		),
		&telebot.SendOptions{ParseMode: telebot.ModeMarkdown},
	)
}

func (b *Bot) handleAlerts(message telebot.Message) {
	alerts, err := listAlerts(b.Config.AlertmanagerURL)
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to list alerts... %v", err), nil)
		return
	}

	if len(alerts) == 0 {
		b.telegram.SendMessage(message.Chat, "No alerts right now! 🎉", nil)
		return
	}

	var out string
	for _, a := range alerts {
		out = out + AlertMessage(a) + "\n"
	}

	b.telegram.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (b *Bot) handleSilences(message telebot.Message) {
	silences, err := listSilences(b.Config.AlertmanagerURL)
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to list silences... %v", err), nil)
		return
	}

	if len(silences) == 0 {
		b.telegram.SendMessage(message.Chat, "No silences right now.", nil)
		return
	}

	var out string
	for _, silence := range silences {
		out = out + SilenceMessage(silence) + "\n"
	}

	b.telegram.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}
