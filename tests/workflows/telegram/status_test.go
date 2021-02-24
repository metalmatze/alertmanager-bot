package telegram

import (
	"fmt"
	"time"

	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var statusWorkflows = []workflow{{
	name: "Status",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStatus,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "*AlertManager*\nVersion: alertmanager\nUptime: 1 minute\n*AlertManager Bot*\nVersion: bot\nUptime: 1 minute",
	}},
	logs: []string{
		"level=debug msg=\"message received\" text=/status",
	},
	alertmanagerStatus: func() string {
		return fmt.Sprintf(
			`{"uptime":%q,"versionInfo":{"version":"alertmanager"}}`,
			time.Now().Add(-time.Minute).Format(time.RFC3339),
		)
	},
}}
