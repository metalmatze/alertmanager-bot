package bot

import "github.com/tucnak/telebot"

// Context is used by Handlers to communicate with Brokers
type Context interface {
	User() telebot.User
	String(msg string) error
	Markdown(msg string) error
}
