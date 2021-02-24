package telegram

import (
	"time"

	"github.com/metalmatze/alertmanager-bot/pkg/alertmanager"
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var webhookWorkflows = []workflow{{
	name:     "WebhookNoSubscribers",
	messages: []telebot.Update{},
	replies:  []reply{},
	logs: []string{
		"level=warn msg=\"chat is not subscribed for alerts\" chat_id=132461234 err=\"chat not found in store\"",
	},
	webhooks: func() []alertmanager.TelegramWebhook {
		webhookFiring.Alerts[0].StartsAt = time.Now().Add(-time.Hour)
		return []alertmanager.TelegramWebhook{{ChatID: 132461234, Message: webhookFiring}}
	},
}, {
	name: "WebhookAdminSubscriber",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStart,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "Hey, Elliot! I will now keep you up to date!\n/help",
	}, {
		recipient: "123",
		message:   "🔥 <b>fire</b> 🔥\n<b>Labels:</b>\n    severity: critical\n<b>Annotations:</b>\n    message: Something is on fire\n<b>Duration:</b> 1 hour",
	}},
	counter: map[string]uint{telegram.CommandStart: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=123",
	},
	webhooks: func() []alertmanager.TelegramWebhook {
		webhookFiring.Alerts[0].StartsAt = time.Now().Add(-time.Hour)
		return []alertmanager.TelegramWebhook{{ChatID: int64(admin.ID), Message: webhookFiring}}
	},
}, {
	name: "WebhookGroupSubscriber",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat: &telebot.Chat{
				ID:   -1234,
				Type: telebot.ChatGroup,
			},
			Text: telegram.CommandStart,
		},
	}},
	replies: []reply{{
		recipient: "-1234",
		message:   "Hey! I will now keep you all up to date!\n/help",
	}, {
		recipient: "-1234",
		message:   "🔥 <b>fire</b> 🔥\n<b>Labels:</b>\n    severity: critical\n<b>Annotations:</b>\n    message: Something is on fire\n<b>Duration:</b> 1 hour",
	}},
	counter: map[string]uint{telegram.CommandStart: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=-1234",
	},
	webhooks: func() []alertmanager.TelegramWebhook {
		webhookFiring.Alerts[0].StartsAt = time.Now().Add(-time.Hour)
		return []alertmanager.TelegramWebhook{{ChatID: int64(-1234), Message: webhookFiring}}
	},
}, {
	name: "WebhookMultipleGroupsSubscribed",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat: &telebot.Chat{
				ID:   -1234,
				Type: telebot.ChatGroup,
			},
			Text: telegram.CommandStart,
		},
	}, {
		Message: &telebot.Message{
			Sender: admin,
			Chat: &telebot.Chat{
				ID:   -5678,
				Type: telebot.ChatGroup,
			},
			Text: telegram.CommandStart,
		},
	}},
	replies: []reply{{
		recipient: "-1234",
		message:   "Hey! I will now keep you all up to date!\n/help",
	}, {
		recipient: "-5678",
		message:   "Hey! I will now keep you all up to date!\n/help",
	}, {
		recipient: "-1234",
		message:   "🔥 <b>fire</b> 🔥\n<b>Labels:</b>\n    severity: critical\n<b>Annotations:</b>\n    message: Something is on fire\n<b>Duration:</b> 1 hour",
	}},
	counter: map[string]uint{telegram.CommandStart: 2},
	logs: []string{
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=-1234",
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=-5678",
	},
	webhooks: func() []alertmanager.TelegramWebhook {
		webhookFiring.Alerts[0].StartsAt = time.Now().Add(-time.Hour)
		return []alertmanager.TelegramWebhook{{ChatID: int64(-1234), Message: webhookFiring}}
	},
}}
