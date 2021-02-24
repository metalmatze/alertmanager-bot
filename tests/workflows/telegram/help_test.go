package telegram

import (
	"strings"

	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var helpWorkflows = []workflow{{
	name: "Help",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandHelp,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   strings.TrimSpace(telegram.ResponseHelp),
	}},
	counter: map[string]uint{telegram.CommandHelp: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/help",
	},
}, {
	name: "HelpAsNobody",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: nobody,
			Chat:   chatFromUser(nobody),
			Text:   telegram.CommandHelp,
		},
	}},
	replies: []reply{},
	logs: []string{
		"level=info msg=\"dropping message from forbidden sender\" sender_id=222 sender_username=nobody",
	},
}}
