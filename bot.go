package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hako/durafmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tucnak/telebot"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"
	commandChats = "/chats"

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

// BotChatStore is all the Bot needs to store and read
type BotChatStore interface {
	List() ([]telebot.Chat, error)
	Add(telebot.Chat) error
	Remove(telebot.Chat) error
}

// Bot runs the alertmanager telegram
type Bot struct {
	logger          log.Logger
	telegram        *telebot.Bot
	chats           BotChatStore
	addr            string
	admin           int
	alertmanager    *url.URL
	commandsCounter *prometheus.CounterVec
	webhooksCounter prometheus.Counter
}

// NewBot creates a Bot with the UserStore and telegram telegram
func NewBot(chats BotChatStore, addr string, alertmanager *url.URL, telegramToken string, telegramAdmin int, opts ...func(b *Bot)) (*Bot, error) {
	bot, err := telebot.NewBot(telegramToken)
	if err != nil {
		return nil, err
	}

	commandsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alertmanagerbot",
		Name:      "commands_total",
		Help:      "Number of commands received by command name",
	}, []string{"command"})
	webhooksCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "alertmanagerbot",
		Name:      "webhooks_total",
		Help:      "Number of webhooks received by this bot",
	})
	prometheus.MustRegister(commandsCounter, webhooksCounter)

	b := &Bot{
		logger:          log.NewNopLogger(),
		telegram:        bot,
		chats:           chats,
		addr:            addr,
		admin:           telegramAdmin,
		alertmanager:    alertmanager,
		commandsCounter: commandsCounter,
		webhooksCounter: webhooksCounter,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

func BotLogger(l log.Logger) func(b *Bot) {
	return func(b *Bot) {
		b.logger = l
	}
}

// RunWebserver starts a http server and listens for messages to send to the users
func (b *Bot) RunWebserver() {
	messages := make(chan string, 100)

	http.HandleFunc("/", HandleWebhook(b.logger, b.webhooksCounter, messages))
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/healthz", handleHealth)

	go b.sendWebhook(messages)

	err := http.ListenAndServe(b.addr, nil)
	level.Error(b.logger).Log("err", err)
	os.Exit(1)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// sendWebhook sends messages received via webhook to all subscribed users
func (b *Bot) sendWebhook(messages <-chan string) {
	for m := range messages {
		chats, err := b.chats.List()
		if err != nil {
			level.Error(b.logger).Log("msg", "failed to get chat list from store", "err", err)
			continue
		}

		for _, chat := range chats {
			b.telegram.SendMessage(chat, m, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		}
	}
}

// SendAdminMessage to the admin's ID with a message
func (b *Bot) SendAdminMessage(adminID int, message string) {
	b.telegram.SendMessage(telebot.User{ID: adminID}, message, nil)
}

// Run the telegram and listen to messages send to the telegram
func (b *Bot) Run() {
	commandSuffix := fmt.Sprintf("@%s", b.telegram.Identity.Username)

	commands := map[string]func(message telebot.Message){
		commandStart:    b.handleStart,
		commandStop:     b.handleStop,
		commandHelp:     b.handleHelp,
		commandChats:    b.handleChats,
		commandStatus:   b.handleStatus,
		commandAlerts:   b.handleAlerts,
		commandSilences: b.handleSilences,
	}

	// init counters with 0
	for command := range commands {
		b.commandsCounter.WithLabelValues(command).Add(0)
	}

	messages := make(chan telebot.Message, 100)
	b.telegram.Listen(messages, time.Second)

	for message := range messages {
		if message.Sender.ID != b.admin {
			b.commandsCounter.WithLabelValues("dropped").Inc()
			level.Info(b.logger).Log(
				"msg", "dropped message from forbidden sender",
				"sender_id", message.Sender.ID,
				"sender_username", message.Sender.Username,
			)
			continue
		}

		b.telegram.SendChatAction(message.Chat, telebot.Typing)

		// Remove the command suffix from the text, /help@BotName => /help
		text := strings.Replace(message.Text, commandSuffix, "", -1)
		level.Debug(b.logger).Log("msg", "message received", "text", text)

		// Get the corresponding handler from the map by the commands text
		if handler, ok := commands[text]; ok {
			b.commandsCounter.WithLabelValues(text).Inc()
			handler(message)
		} else {
			b.commandsCounter.WithLabelValues("incomprehensible").Inc()
			b.telegram.SendMessage(
				message.Chat,
				"Sorry, I don't understand...",
				nil,
			)
		}
	}
}

func (b *Bot) handleStart(message telebot.Message) {
	if err := b.chats.Add(message.Chat); err != nil {
		level.Warn(b.logger).Log("msg", "failed to add chat to chat store", "err", err)
		b.telegram.SendMessage(message.Sender, "I can't add this chat to the subscribers list, see logs", nil)
		return
	}

	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
	level.Info(b.logger).Log(
		"user subscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
}

func (b *Bot) handleStop(message telebot.Message) {
	if err := b.chats.Remove(message.Chat); err != nil {
		level.Warn(b.logger).Log("msg", "failed to remove chat from chat store", "err", err)
		b.telegram.SendMessage(message.Sender, "I can't remove this chat from the subscribers list, see logs", nil)
		return
	}

	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
	level.Info(b.logger).Log(
		"user unsubscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
}

func (b *Bot) handleHelp(message telebot.Message) {
	b.telegram.SendMessage(message.Chat, responseHelp, nil)
}

func (b *Bot) handleChats(message telebot.Message) {
	chats, err := b.chats.List()
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to remove chat from chat store", "err", err)
		b.telegram.SendMessage(message.Sender, "I can't remove this chat from the subscribers list, see logs", nil)
		return
	}

	list := ""
	for _, chat := range chats {
		if chat.IsGroupChat() {
			list = list + fmt.Sprintf("@%s\n", chat.Title)
		} else {
			list = list + fmt.Sprintf("@%s\n", chat.Username)
		}
	}

	b.telegram.SendMessage(message.Chat, "Currently these chat have subscribed:\n"+list, nil)
}

func (b *Bot) handleStatus(message telebot.Message) {
	s, err := status(b.logger, b.alertmanager.String())
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
			Revision,
			uptimeBot,
		),
		&telebot.SendOptions{ParseMode: telebot.ModeMarkdown},
	)
}

func (b *Bot) handleAlerts(message telebot.Message) {
	alerts, err := listAlerts(b.logger, b.alertmanager.String())
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
	silences, err := listSilences(b.logger, b.alertmanager.String())
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
