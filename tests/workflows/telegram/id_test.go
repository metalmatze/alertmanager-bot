package telegram

import (
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var idWorkflows = []workflow{{
	name: "IDAsNobody",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: nobody,
			Chat:   chatFromUser(nobody),
			Text:   telegram.CommandID,
		},
	}},
	replies: []reply{{
		recipient: "222",
		message:   "Your ID is 222",
	}},
	counter: map[string]uint{telegram.CommandID: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/id",
	},
}, {
	name: "IDAsAdmin",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandID,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "Your ID is 123",
	}},
	counter: map[string]uint{telegram.CommandID: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/id",
	},
}}
