package telegram

import (
	"fmt"
	"time"

	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"gopkg.in/tucnak/telebot.v2"
)

var alertsWorkflows = []workflow{{
	name: "AlertsNone",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "No alerts right now! 🎉",
	}},
	logs: []string{
		"level=debug msg=\"message received\" text=/alerts",
	},
}, {
	name: "AlertsFiring",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "🔥 <b>damn</b> 🔥\n<b>Labels:</b>\n    bot: alertmanager-bot\n<b>Annotations:</b>\n    msg: sup?!\n    runbook: https://example.com/runbook\n<b>Duration:</b> 1 hour",
	}},
	logs: []string{
		"level=debug msg=\"message received\" text=/alerts",
	},
	alertmanagerAlerts: func() string {
		return fmt.Sprintf(
			`[{"labels":{"alertname":"damn","bot":"alertmanager-bot"},"annotations":{"msg":"sup?!","runbook":"https://example.com/runbook"},"startsAt":"%s"}]`,
			time.Now().Add(-time.Hour).Format(time.RFC3339),
		)
	},
}, {
	name: "AlertsResolved",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "✅ <b>damn</b> ✅\n<b>Labels:</b>\n    bot: alertmanager-bot\n<b>Annotations:</b>\n    msg: sup?!\n<b>Duration:</b> 58 minutes\n<b>Ended:</b> 2 minutes",
	}},
	logs: []string{
		"level=debug msg=\"message received\" text=/alerts",
	},
	alertmanagerAlerts: func() string {
		return fmt.Sprintf(
			`[{"labels":{"alertname":"damn","bot":"alertmanager-bot"},"annotations":{"msg":"sup?!"},"startsAt": "%s","endsAt": "%s"}]`,
			time.Now().Add(-time.Hour).Format(time.RFC3339),
			time.Now().Add(-2*time.Minute).Format(time.RFC3339),
		)
	},
}}
