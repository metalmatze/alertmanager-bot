package telegram

import (
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var chatsWorkflows = []workflow{{
	name: "ChatsNone",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandChats,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "Currently no one is subscribed.",
	}},
	counter: map[string]uint{telegram.CommandChats: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/chats",
	},
}, {
	name: "ChatsWithAdminSubscribed",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStart,
		},
	}, {
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandChats,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "Hey, Elliot! I will now keep you up to date!\n/help",
	}, {
		recipient: "123",
		message:   "Currently these chat have subscribed:\n@elliot",
	}},
	counter: map[string]uint{
		telegram.CommandChats: 1,
		telegram.CommandStart: 1,
	},
	logs: []string{
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=123",
		"level=debug msg=\"message received\" text=/chats",
	},
}, {
	name: "ChatsWithNoUsernameSubscribed",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: anonymousAdmin,
			Chat:   chatFromUser(anonymousAdmin),
			Text:   telegram.CommandStart,
		},
	}, {
		Message: &telebot.Message{
			Sender: anonymousAdmin,
			Chat:   chatFromUser(anonymousAdmin),
			Text:   telegram.CommandChats,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "Hey! I will now keep you up to date!\n/help",
	}, {
		recipient: "123",
		message:   "Currently these chat have subscribed:\n@123",
	}},
	counter: map[string]uint{
		telegram.CommandChats: 1,
		telegram.CommandStart: 1,
	},
	logs: []string{
		"level=debug msg=\"message received\" text=/start",
		"level=info msg=\"user subscribed\" username= user_id=123 chat_id=123",
		"level=debug msg=\"message received\" text=/chats",
	},
}}
