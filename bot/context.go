package bot

import "github.com/tucnak/telebot"

// Context is passed to all handlers and allows to pass data between those in a chain
type Context struct {
	// TODO
	//ctx     context.Context
	engine  *Engine
	message telebot.Message
}

// User returns the user of the incoming message
func (c *Context) User() telebot.User {
	return c.message.Sender
}

// String sends a string back to the user
func (c *Context) String(msg string) error {
	return c.engine.telegram.SendMessage(c.message.Chat, msg, nil)
}

// Markdown sends a markdown formatted string back to the user
func (c *Context) Markdown(msg string) error {
	options := &telebot.SendOptions{ParseMode: telebot.ModeMarkdown}
	return c.engine.telegram.SendMessage(c.message.Chat, msg, options)
}
