package telegram

import (
	"fmt"
	"strings"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/notify/webhook"
	"gopkg.in/tucnak/telebot.v2"
)

// sendWebhook sends messages received via webhook to all subscribed chats
func (b *Bot) sendWebhook(message webhook.Message) error {
	chats, err := b.store.List()
	if err != nil {
		return fmt.Errorf("failed to get chat list from store: %w", err)
	}

	out, err := b.templates.ExecuteHTMLString(`{{ template "telegram.default" . }}`, message.Data)
	if err != nil {
		return fmt.Errorf("failed to template alerts: %w", err)
	}

	for _, chat := range chats {
		_, err = b.telebot.Send(chat, b.truncateMessage(out), &telebot.SendOptions{ParseMode: telebot.ModeHTML})
		if err != nil {
			return fmt.Errorf("failed to send message to subscribed chat: %w", err)
		}
	}

	return nil
}

// Truncate very big message
func (b *Bot) truncateMessage(str string) string {
	truncateMsg := str
	if len(str) > 4095 { // telegram API can only support 4096 bytes per message
		level.Warn(b.logger).Log("msg", "message is bigger than 4095, truncate...")
		// find the end of last alert, we do not want break the html tags
		i := strings.LastIndex(str[0:4080], "\n\n") // 4080 + "\n<b>[SNIP]</b>" == 4095
		if i > 1 {
			truncateMsg = str[0:i] + "\n<b>[SNIP]</b>"
		} else {
			truncateMsg = "Message is too long... can't send.."
			level.Warn(b.logger).Log("msg", "truncateMessage: Unable to find the end of last alert.")
		}
		return truncateMsg
	}
	return truncateMsg
}
