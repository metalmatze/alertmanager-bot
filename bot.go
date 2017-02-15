package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-kit/kit/log/levels"
	"github.com/hako/durafmt"
	"github.com/metalmatze/alertmanager-bot/bot"
	"github.com/prometheus/client_golang/prometheus"
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

// AlertmanagerBot runs the alertmanager telegram
type AlertmanagerBot struct {
	logger    levels.Levels
	Config    Config
	UserStore *UserStore
}

// NewAlertmanagerBot creates a AlertmanagerBot with the UserStore and telegram telegram
func NewAlertmanagerBot(logger levels.Levels, c Config) (*AlertmanagerBot, error) {
	users, err := NewUserStore(c.Store)
	if err != nil {
		return nil, err
	}

	return &AlertmanagerBot{
		logger:    logger,
		Config:    c,
		UserStore: users,
	}, nil
}

// RunWebserver starts a http server and listens for messages to send to the users
func (b *AlertmanagerBot) RunWebserver() {
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
func (b *AlertmanagerBot) sendWebhook(messages <-chan string) {
	//for m := range messages {
	//	for _, user := range b.UserStore.List() {
	//		b.telegram.SendMessage(user, m, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
	//	}
	//}
}

func (b *AlertmanagerBot) handleStart(c *bot.Context) error {
	if err := b.UserStore.Add(c.User()); err != nil {
		b.logger.Error().Log(
			"msg", "can't remove user from store",
			"err", err,
		)
		return fmt.Errorf("can't remove user %s from store", c.User().Username)
	}

	b.logger.Info().Log(
		"user subscribed",
		"username", c.User().Username,
		"user_id", c.User().ID,
	)

	return c.String(fmt.Sprintf(responseStart, c.User().FirstName))
}

func (b *AlertmanagerBot) handleStop(c *bot.Context) error {
	if err := b.UserStore.Remove(c.User()); err != nil {
		b.logger.Error().Log(
			"msg", "can't remove user from store",
			"err", err,
		)
		return fmt.Errorf("can't remove user %s from store", c.User().Username)
	}
	b.logger.Info().Log(
		"user unsubscribed",
		"username", c.User().Username,
		"user_id", c.User().ID,
	)

	return c.String(fmt.Sprintf(responseStop, c.User().FirstName))
}

func (b *AlertmanagerBot) handleHelp(c *bot.Context) error {
	return c.String(responseHelp)
}

func (b *AlertmanagerBot) handleUsers(c *bot.Context) error {
	return c.String(fmt.Sprintf("Currently %d users are subscribed.", b.UserStore.Len()))
}

func (b *AlertmanagerBot) handleStatus(c *bot.Context) error {
	s, err := status(b.logger, b.Config.AlertmanagerURL)
	if err != nil {
		return fmt.Errorf("failed to get status: %v", err)
	}

	uptime := durafmt.Parse(time.Since(s.Data.Uptime))
	uptimeBot := durafmt.Parse(time.Since(StartTime))

	message := fmt.Sprintf(
		"*AlertManager*\nVersion: %s\nUptime: %s\n*AlertManager Bot*\nVersion: %s\nUptime: %s",
		s.Data.VersionInfo.Version,
		uptime,
		Commit,
		uptimeBot,
	)

	return c.Markdown(message)
}

func (b *AlertmanagerBot) handleAlerts(c *bot.Context) error {
	alerts, err := listAlerts(b.logger, b.Config.AlertmanagerURL)
	if err != nil {
		return fmt.Errorf("failed to list alerts: %v", err)
	}

	if len(alerts) == 0 {
		return c.String("No alerts right now! 🎉")
	}

	var out string
	for _, a := range alerts {
		out = out + AlertMessage(a) + "\n"
	}

	return c.Markdown(out)
}

func (b *AlertmanagerBot) handleSilences(c *bot.Context) error {
	silences, err := listSilences(b.logger, b.Config.AlertmanagerURL)
	if err != nil {
		return fmt.Errorf("failed to list silences: %v", err)
	}

	if len(silences) == 0 {
		return c.String("No silences right now.")
	}

	var out string
	for _, silence := range silences {
		out = out + SilenceMessage(silence) + "\n"
	}

	return c.Markdown(out)
}
