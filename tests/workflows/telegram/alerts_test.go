package telegram

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"github.com/stretchr/testify/require"
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
	counter: map[string]uint{telegram.CommandAlerts: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/alerts",
	},
	alertmanagerStatus: func(t *testing.T, r *http.Request) string {
		return `{"config":{"original":"route:\n  receiver: admin\nreceivers:\n- name: admin\n  webhook_configs:\n  - send_resolved: true\n    url: http://localhost:8080/webhooks/telegram/123"}}`
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
	counter: map[string]uint{telegram.CommandAlerts: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/alerts",
	},
	alertmanagerAlerts: func(t *testing.T, r *http.Request) string {
		require.Equal(t, "true", r.URL.Query().Get("active"))
		require.Equal(t, "true", r.URL.Query().Get("inhibited"))
		require.Equal(t, "admin", r.URL.Query().Get("receiver"))
		require.Equal(t, "false", r.URL.Query().Get("silenced"))
		require.Equal(t, "true", r.URL.Query().Get("unprocessed"))

		return fmt.Sprintf(
			`[{"labels":{"alertname":"damn","bot":"alertmanager-bot"},"annotations":{"msg":"sup?!","runbook":"https://example.com/runbook"},"startsAt":"%s"}]`,
			time.Now().Add(-time.Hour).Format(time.RFC3339),
		)
	},
	alertmanagerStatus: func(t *testing.T, r *http.Request) string {
		return `{"config":{"original":"route:\n  receiver: admin\nreceivers:\n- name: admin\n  webhook_configs:\n  - send_resolved: true\n    url: http://localhost:8080/webhooks/telegram/123"}}`
	},
}, {
	name: "AlertsFiringSilenced",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts + " silenced",
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "🔥 <b>damn</b> 🔥\n<b>Labels:</b>\n    bot: alertmanager-bot\n<b>Annotations:</b>\n    msg: sup?!\n    runbook: https://example.com/runbook\n<b>Duration:</b> 1 hour",
	}},
	counter: map[string]uint{telegram.CommandAlerts: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=\"/alerts silenced\"",
	},
	alertmanagerAlerts: func(t *testing.T, r *http.Request) string {
		require.Equal(t, "true", r.URL.Query().Get("active"))
		require.Equal(t, "true", r.URL.Query().Get("inhibited"))
		require.Equal(t, "admin", r.URL.Query().Get("receiver"))
		require.Equal(t, "true", r.URL.Query().Get("silenced"))
		require.Equal(t, "true", r.URL.Query().Get("unprocessed"))

		return fmt.Sprintf(
			`[{"labels":{"alertname":"damn","bot":"alertmanager-bot"},"annotations":{"msg":"sup?!","runbook":"https://example.com/runbook"},"startsAt":"%s"}]`,
			time.Now().Add(-time.Hour).Format(time.RFC3339),
		)
	},
	alertmanagerStatus: func(t *testing.T, r *http.Request) string {
		return `{"config":{"original":"route:\n  receiver: admin\nreceivers:\n- name: admin\n  webhook_configs:\n  - send_resolved: true\n    url: http://localhost:8080/webhooks/telegram/123"}}`
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
	counter: map[string]uint{telegram.CommandAlerts: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/alerts",
	},
	alertmanagerAlerts: func(t *testing.T, r *http.Request) string {
		require.Equal(t, "true", r.URL.Query().Get("active"))
		require.Equal(t, "true", r.URL.Query().Get("inhibited"))
		require.Equal(t, "admin", r.URL.Query().Get("receiver"))
		require.Equal(t, "false", r.URL.Query().Get("silenced"))
		require.Equal(t, "true", r.URL.Query().Get("unprocessed"))

		return fmt.Sprintf(
			`[{"labels":{"alertname":"damn","bot":"alertmanager-bot"},"annotations":{"msg":"sup?!"},"startsAt": "%s","endsAt": "%s"}]`,
			time.Now().Add(-time.Hour).Format(time.RFC3339),
			time.Now().Add(-2*time.Minute).Format(time.RFC3339),
		)
	},
	alertmanagerStatus: func(t *testing.T, r *http.Request) string {
		return `{"config":{"original":"route:\n  receiver: admin\nreceivers:\n- name: admin\n  webhook_configs:\n  - send_resolved: true\n    url: http://localhost:8080/webhooks/telegram/123"}}`
	},
}, {
	name: "AlertsNotSetup",
	messages: []telebot.Update{{
		Message: &telebot.Message{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts,
		},
	}},
	replies: []reply{{
		recipient: "123",
		message:   "This chat hasn't been setup to receive any alerts yet... 😕\n\nAsk an administrator of the Alertmanager to add a webhook with `/webhooks/telegram/123` as URL.",
	}},
	counter: map[string]uint{telegram.CommandAlerts: 1},
	logs: []string{
		"level=debug msg=\"message received\" text=/alerts",
	},
	alertmanagerStatus: func(t *testing.T, r *http.Request) string {
		return `{"config":{"original":"route:\n  receiver: admin\nreceivers:\n- name: admin\n  webhook_configs:\n  - send_resolved: true\n    url: http://localhost:8080/webhooks/telegram/unknown"}}`
	},
}}
