package bot

import (
	"time"

	"github.com/tucnak/telebot"
)

// HandleFunc is used to generate the response to a request
type HandleFunc func(*Context) error

// HandlerChain is the chain used for by command
type HandlerChain []HandleFunc

// Engine is the foundation for the bot
// Create a new one by using New()
type Engine struct {
	telegram *telebot.Bot
	commands map[string]HandlerChain
}

// New creates a new bot Engine
func New(Token string) (*Engine, error) {
	telegram, err := telebot.NewBot(Token)
	if err != nil {
		return nil, err
	}

	return &Engine{
		telegram: telegram,
		commands: make(map[string]HandlerChain),
	}, nil
}

// Run the telegram and listen to messages send to the telegram
func (e *Engine) Run() error {
	messages := make(chan telebot.Message, 100)
	e.telegram.Listen(messages, time.Second)

	for message := range messages {
		e.telegram.SendChatAction(message.Chat, telebot.Typing)

		ctx := &Context{engine: e, message: message}

		if handlers, ok := e.commands[message.Text]; ok {
			for _, handler := range handlers {
				if err := handler(ctx); err != nil {
					e.telegram.SendMessage(message.Chat, err.Error(), nil)
					break
				}
			}
		}
	}

	return nil
}

// HandleFunc registers the handler function for the given command
func (e *Engine) HandleFunc(command string, handlers ...HandleFunc) {
	e.commands[command] = handlers
}
