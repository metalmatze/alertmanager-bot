package telegram

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
)

type Alertmanager interface {
	ListAlerts() ([]*types.Alert, error)
}

type Bot struct {
	logger       log.Logger
	telebot      *telebot.Bot
	alertmanager Alertmanager

	templates *template.Template
}

func NewBot(logger log.Logger, am Alertmanager, templates *template.Template, token string) (*Bot, error) {
	t, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	b := &Bot{logger: logger, telebot: t, alertmanager: am, templates: templates}

	t.Handle(commandStart, b.handler(b.handleStart))
	t.Handle(commandAlerts, b.handler(b.handleAlerts))
	t.Handle(telebot.OnText, b.handler(b.handleDefault))

	return b, nil
}

func (b *Bot) Run() {
	b.telebot.Start()
}

func (b *Bot) Shutdown() {
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
	_, err := b.telebot.Send(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName))
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
