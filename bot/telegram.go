package bot

import (
	"time"

	"github.com/tucnak/telebot"
)

// TelegramBroker implements the Broker interface and
// allows communication between the bot and telegram
type TelegramBroker struct {
	engine   *Engine
	telegram *telebot.Bot
}

// NewTelegramBroker returns a TelegramBroker that's connected to telegram
func NewTelegramBroker(e *Engine, Token string) (*TelegramBroker, error) {
	telegram, err := telebot.NewBot(Token)
	if err != nil {
		return nil, err
	}

	return &TelegramBroker{
		engine:   e,
		telegram: telegram,
	}, nil
}

// Name returns the name of the Broker: telegram
func (b *TelegramBroker) Name() string {
	return "telegram"
}

// Run the TelegramBroker and receive incoming messages via channel
func (b *TelegramBroker) Run(in chan<- Context) {
	messages := make(chan telebot.Message, 100)
	b.telegram.Listen(messages, time.Second)

	for message := range messages {
		b.telegram.SendChatAction(message.Chat, telebot.Typing)

		//ctx := &Context{engine: e, message: message}
		ctx := &TelegramContext{broker: b, message: message}

		if handlers, ok := b.engine.commands[message.Text]; ok {
			for _, handler := range handlers {
				if err := handler(ctx); err != nil {
					b.telegram.SendMessage(message.Chat, err.Error(), nil)
					break
				}
			}
		}
	}

}

// TelegramContext implements the Context interface and
// makes sure everything is passed on to telegram
type TelegramContext struct {
	// TODO
	//ctx     context.Context
	broker  *TelegramBroker
	message telebot.Message
}

// User returns the user of the incoming message
func (c *TelegramContext) Broker() string {
	return c.broker.Name()
}

// User returns the user of the incoming message
func (c *TelegramContext) User() telebot.User {
	return c.message.Sender
}

// String sends a string back to the user
func (c *TelegramContext) String(msg string) error {
	return c.broker.telegram.SendMessage(c.message.Chat, msg, nil)
}

// Markdown sends a markdown formatted string back to the user
func (c *TelegramContext) Markdown(msg string) error {
	options := &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}
	return c.broker.telegram.SendMessage(c.message.Chat, msg, options)
}
