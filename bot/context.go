package bot

import "github.com/tucnak/telebot"

// Context is used by Handlers to communicate with Brokers
type Context interface {
	Broker() string
	Raw() string
	User() telebot.User

	// Write
	String(msg string) error
	Markdown(msg string) error
}
