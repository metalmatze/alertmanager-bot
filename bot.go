package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-kit/kit/log/levels"
	"github.com/hako/durafmt"
	"github.com/prometheus/client_golang/prometheus"
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

var (
	commandsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alertmanagerbot",
		Name:      "commands_total",
		Help:      "Number of commands received by command name",
	}, []string{"command"})
	webhooksCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "alertmanagerbot",
		Name:      "webhooks_total",
		Help:      "Number of webhooks received by this bot",
	})
)

func init() {
	prometheus.MustRegister(commandsCounter, webhooksCounter)
}

// HandleFunc is used to generate the response to a request
type HandleFunc func(telebot.Message) error

// Bot runs the alertmanager telegram
type Bot struct {
	logger    levels.Levels
	telegram  *telebot.Bot
	commands  map[string][]HandleFunc
	Config    Config
	UserStore *UserStore
}

// NewBot creates a Bot with the UserStore and telegram telegram
func NewBot(logger levels.Levels, c Config) (*Bot, error) {
	users, err := NewUserStore(c.Store)
	if err != nil {
		return nil, err
	}

	bot, err := telebot.NewBot(c.TelegramToken)
	if err != nil {
		return nil, err
	}

	return &Bot{
		logger:    logger,
		telegram:  bot,
		commands:  make(map[string][]HandleFunc),
		Config:    c,
		UserStore: users,
	}, nil
}

// RunWebserver starts a http server and listens for messages to send to the users
func (b *Bot) RunWebserver() {
	messages := make(chan string, 100)

	http.HandleFunc("/", HandleWebhook(messages))
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/healthz", handleHealth)

	go b.sendWebhook(messages)

	err := http.ListenAndServe(b.Config.ListenAddr, nil)
	b.logger.Crit().Log("err", err)
	os.Exit(1)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// sendWebhook sends messages received via webhook to all subscribed users
func (b *Bot) sendWebhook(messages <-chan string) {
	for m := range messages {
		for _, user := range b.UserStore.List() {
			b.telegram.SendMessage(user, m, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		}
	}
}

// HandleFunc registers the handler function for the given command
func (b *Bot) HandleFunc(command string, handlers ...HandleFunc) {
	b.commands[command] = handlers
}

// Run the telegram and listen to messages send to the telegram
func (b *Bot) Run() {
	messages := make(chan telebot.Message, 100)
	b.telegram.Listen(messages, time.Second)

	for message := range messages {
		if message.Sender.ID != b.Config.TelegramAdmin {
			commandsCounter.WithLabelValues("dropped").Inc()
			b.logger.Info().Log(
				"msg", "dropped message from unallowed sender",
				"sender_id", message.Sender.ID,
				"sender_username", message.Sender.Username,
			)
			continue
		}

		b.telegram.SendChatAction(message.Chat, telebot.Typing)

		if handlers, ok := b.commands[message.Text]; ok {
			for _, handler := range handlers {
				if err := handler(message); err != nil {
					b.logger.Info().Log(
						"msg", "handler returned err",
						"err", err,
					)
					b.telegram.SendMessage(message.Chat, err.Error(), nil)
					continue
				}
			}
		}
	}
}

func (b *Bot) instrument(message telebot.Message) error {
	command := message.Text
	if _, ok := b.commands[command]; ok {
		commandsCounter.WithLabelValues(command).Inc()
		return nil
	}

	commandsCounter.WithLabelValues("incomprehensible").Inc()
	return b.telegram.SendMessage(
		message.Chat,
		"Sorry, I don't understand...",
		nil,
	)
}

func (b *Bot) handleStart(message telebot.Message) error {
	if err := b.UserStore.Add(message.Sender); err != nil {
		b.logger.Error().Log(
			"msg", "can't remove user from store",
			"err", err,
		)
		return fmt.Errorf("can't remove user %s from store", message.Sender.Username)
	}

	b.logger.Info().Log(
		"user subscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)

	return b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
}

func (b *Bot) handleStop(message telebot.Message) error {
	if err := b.UserStore.Remove(message.Sender); err != nil {
		b.logger.Error().Log(
			"msg", "can't remove user from store",
			"err", err,
		)
		return fmt.Errorf("can't remove user %s from store", message.Sender.Username)
	}
	b.logger.Info().Log(
		"user unsubscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)

	return b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
}

func (b *Bot) handleHelp(message telebot.Message) error {
	return b.telegram.SendMessage(message.Chat, responseHelp, nil)
}

func (b *Bot) handleUsers(message telebot.Message) error {
	return b.telegram.SendMessage(message.Chat, fmt.Sprintf(
		"Currently %d users are subscribed.",
		b.UserStore.Len()),
		nil,
	)
}

func (b *Bot) handleStatus(message telebot.Message) error {
	s, err := status(b.logger, b.Config.AlertmanagerURL)
	if err != nil {
		return fmt.Errorf("failed to get status: %v", err)
	}

	uptime := durafmt.Parse(time.Since(s.Data.Uptime))
	uptimeBot := durafmt.Parse(time.Since(StartTime))

	return b.telegram.SendMessage(
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

func (b *Bot) handleAlerts(message telebot.Message) error {
	alerts, err := listAlerts(b.logger, b.Config.AlertmanagerURL)
	if err != nil {
		return fmt.Errorf("failed to list alerts: %v", err)
	}

	if len(alerts) == 0 {
		return b.telegram.SendMessage(message.Chat, "No alerts right now! 🎉", nil)
	}

	var out string
	for _, a := range alerts {
		out = out + AlertMessage(a) + "\n"
	}

	return b.telegram.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (b *Bot) handleSilences(message telebot.Message) error {
	silences, err := listSilences(b.logger, b.Config.AlertmanagerURL)
	if err != nil {
		return fmt.Errorf("failed to list silences: %v", err)
	}

	if len(silences) == 0 {
		return b.telegram.SendMessage(message.Chat, "No silences right now.", nil)
	}

	var out string
	for _, silence := range silences {
		out = out + SilenceMessage(silence) + "\n"
	}

	return b.telegram.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}
