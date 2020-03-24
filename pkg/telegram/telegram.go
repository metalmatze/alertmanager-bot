package telegram

import (
	"fmt"
	"net/url"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hako/durafmt"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"gopkg.in/tucnak/telebot.v2"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"
	//commandChats = "/chats"

	commandAlerts = "/alerts"

	responseStart = "Hey, %s! I will now keep you up to date!\n" + commandHelp
	responseStop  = "Alright, %s! I won't talk to you again.\n" + commandHelp
)

// Store is a combination of different smaller interfaces for telegram storage.
type Store interface {
}

// Alertmanager is the interface describing functions
// the bot needs to communicate with Alertmanager.
type Alertmanager interface {
	ListAlerts() ([]*types.Alert, error)
}

// Teleboter describes the Telegram Bot's needed capabilities
type Teleboter interface {
	Start()
	Stop()
	Send(to telebot.Recipient, what interface{}, options ...interface{}) (*telebot.Message, error)
	Handle(endpoint interface{}, handler interface{})
}

//Bot is the Telegram bot itself. It makes requests to Alertmanager, converts to Telegram
// and handles notifying for incoming webhooks.
type Bot struct {
	logger  log.Logger
	telebot Teleboter
	//telebot      *telebot.Bot
	alertmanager Alertmanager
	store        *ChatStore

	templates *template.Template

	shutdown chan struct{}
}

// BotOption passed to NewBot to change the default instance
type BotOption func(b *Bot)

// NewBot creates a new Telegram Alertmanager Bot.
func NewBot(store Store, am Alertmanager, token string, opts ...BotOption) (*Bot, error) {
	b := &Bot{
		alertmanager: am,
		logger:       log.NewNopLogger(),
		store:        NewChatStore(),
		shutdown:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.telebot == nil {
		t, err := telebot.NewBot(telebot.Settings{
			Token:  token,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bot: %w", err)
		}
		b.telebot = t
	}

	b.telebot.Handle(commandStart, b.handler(b.handleStart))
	b.telebot.Handle(commandStop, b.handler(b.handleStop))
	b.telebot.Handle(commandAlerts, b.handler(b.handleAlerts))
	b.telebot.Handle(telebot.OnText, b.handler(b.handleDefault))

	return b, nil
}

// WithTelebot uses another implementation for a Telegram bot
func WithTelebot(teleboter Teleboter) BotOption {
	return func(b *Bot) {
		b.telebot = teleboter
	}
}

// WithLogger sets the logger for the Bot as an option
func WithLogger(l log.Logger) BotOption {
	return func(b *Bot) {
		b.logger = l
	}
}

// WithTemplate creates a Template from file so that we can template alerts for Telegram.
func WithTemplate(alertmanagerURL *url.URL, paths ...string) BotOption {
	funcs := template.DefaultFuncs
	funcs["since"] = func(t time.Time) string {
		return durafmt.Parse(time.Since(t)).String()
	}
	funcs["duration"] = func(start time.Time, end time.Time) string {
		return durafmt.Parse(end.Sub(start)).String()
	}

	template.DefaultFuncs = funcs
	tmpl, err := template.FromGlobs(paths...)
	if err != nil {
		panic(fmt.Errorf("failed to parse templates: %w", err))
	}

	tmpl.ExternalURL = alertmanagerURL

	return func(b *Bot) {
		b.templates = tmpl
	}
}

// Run the bot.
func (b *Bot) Run(messages <-chan webhook.Message) {
	level.Info(b.logger).Log("msg", "starting Telegram bot")

	go b.telebot.Start()

	for {
		select {
		case <-b.shutdown:
			return
		case message := <-messages:
			b.sendWebhook(message)
		}
	}
}

// Shutdown the bot gracefully.
func (b *Bot) Shutdown() {
	b.shutdown <- struct{}{}

	level.Info(b.logger).Log("msg", "shutting down Telegram bot")
	b.telebot.Stop()
}

type messageHandler func(message *telebot.Message) error

func (b *Bot) handler(next messageHandler) interface{} {
	return func(message *telebot.Message) {
		err := next(message)
		if err != nil {
			level.Warn(b.logger).Log("msg", "failed handling message", "err", err)
		} else {
			level.Debug(b.logger).Log("msg", "handled message", "message", message.Text, "sender", message.Sender.ID)
		}
	}
}

func (b *Bot) handleDefault(message *telebot.Message) error {
	return fmt.Errorf("unknown message: %s by %d", message.Text, message.Sender.ID)
}

func (b *Bot) handleStart(message *telebot.Message) error {
	if err := b.store.Add(message.Chat); err != nil {
		return fmt.Errorf("failed to add chat to store: %w", err)
	}

	_, err := b.telebot.Send(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName))
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) handleStop(message *telebot.Message) error {
	if err := b.store.Remove(message.Chat.ID); err != nil {
		return fmt.Errorf("failed to remove chat from store: %w", err)
	}

	_, err := b.telebot.Send(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName))
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) handleAlerts(message *telebot.Message) error {
	alert, err := b.alertmanager.ListAlerts()
	if err != nil {
		return fmt.Errorf("failed to list alerts: %w", err)
	}

	out, err := b.tmplAlerts(alert...)
	if err != nil {
		return err
	}

	_, err = b.telebot.Send(message.Chat, out, telebot.ModeHTML)
	if err != nil {
		return err
	}

	return nil
}

func (b *Bot) tmplAlerts(alerts ...*types.Alert) (string, error) {
	data := b.templates.Data("default", nil, alerts...)

	out, err := b.templates.ExecuteHTMLString(`{{ template "telegram.default" . }}`, data)
	if err != nil {
		return "", err
	}

	return out, nil
}
