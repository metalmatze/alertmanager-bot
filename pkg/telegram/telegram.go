package telegram

import (
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"gopkg.in/tucnak/telebot.v2"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"
	//commandChats = "/chats"

	responseStart = "Hey, %s! I will now keep you up to date!\n" + commandHelp
)

type Bot struct {
	logger  log.Logger
	telebot *telebot.Bot
}

func NewBot(logger log.Logger, token string) (*Bot, error) {
	t, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	b := &Bot{logger: logger, telebot: t}

	t.Handle(commandStart, b.handler(b.handleStart))
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
