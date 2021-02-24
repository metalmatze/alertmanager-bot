package telegram

import (
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var stopWorkflows = []workflow{{
	name: "StopWithoutStart",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStop,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "Alright, Elliot! I won't talk to you again.\n/help",
	}},
	counter: map[string]uint{telegram.CommandStop: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/stop",
		"level=info msg=\"user unsubscribed\" username=elliot user_id=123",
	},
}}
