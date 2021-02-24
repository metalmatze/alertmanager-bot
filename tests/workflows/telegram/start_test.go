package telegram

import (
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var startWorkflows = []workflow{{
	name: "StartPrivate",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStart,
		}},
	},
	replies: []reply{{
		recipient: "123",
		message:   "Hey, Elliot! I will now keep you up to date!\n/help",
	}},
	counter: map[string]uint{telegram.CommandStart: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=123",
	},
}, {
	name: "StartGroup",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat: &telebot.Chat{
				ID:   -1234,
				Type: telebot.ChatGroup,
			},
			Text: telegram.CommandStart,
		}},
	},
	replies: []reply{{
		recipient: "-1234",
		message:   "Hey! I will now keep you all up to date!\n/help",
	}},
	counter: map[string]uint{telegram.CommandStart: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=-1234",
	},
}}
