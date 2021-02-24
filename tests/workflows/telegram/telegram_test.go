package telegram

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/metalmatze/alertmanager-bot/pkg/alertmanager"
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/require"
	"gopkg.in/tucnak/telebot.v2"
)

var (
	admin = &telebot.User{
		ID:        123,
		FirstName: "Elliot",
		LastName:  "Alderson",
		Username:  "elliot",
		IsBot:     false,
	}
	nobody = &telebot.User{
		ID:        222,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "nobody",
		IsBot:     false,
	}

	webhookFiring = webhook.Message{
		Data: &template.Data{
			Receiver: "telegram",
			Status:   "firing",
			Alerts: template.Alerts{{
				Status:      "firing",
				Labels:      template.KV{"alertname": "fire", "severity": "critical"},
				Annotations: template.KV{"message": "Something is on fire"},
				//StartsAt:     time.Now().Add(-time.Hour),
				EndsAt:       time.Time{},
				GeneratorURL: "http://localhost:9090/graph?g0.expr=vector%28666%29\\u0026g0.tab=1",
				Fingerprint:  "",
			}},
			GroupLabels:       template.KV{"alertname": "Fire"},
			CommonLabels:      template.KV{"alertname": "Fire", "severity": "critical"},
			CommonAnnotations: template.KV{"message": "Something is on fire"},
			ExternalURL:       "http://localhost:9093",
		},
		Version:         "4",
		GroupKey:        `{}:{alertname="Fire"}`,
		TruncatedAlerts: 0,
	}

	// These are the different workflows/scenarios we are testing.
	workflows = []struct {
		name               string
		messages           []telebot.Update
		replies            []reply
		logs               []string
		webhooks           func() []alertmanager.TelegramWebhook
		alertmanagerAlerts func() string
		alertmanagerStatus func() string
	}{{
		name: "Dropped",
		messages: []telebot.Update{{
			Message: &telebot.Message{
				Sender: nobody,
			},
		}},
		replies: []reply{},
		logs: []string{
			"", // TODO: "level=info msg=\"failed to process message\" err=\"dropped message from forbidden sender\" sender_id=222 sender_username=nobody",
		},
	}, {
		name: "Incomprehensible",
		messages: []telebot.Update{{
			Message: &telebot.Message{
				Sender: admin,
				Chat:   chatFromUser(admin),
				Text:   "/incomprehensible",
			},
		}},
		//TODO:
		//replies: []reply{{
		//	recipient: "123",
		//	message:   "Sorry, I don't understand...",
		//}},
		logs: []string{
			"", //TODO: "level=debug msg=\"message received\" text=/incomprehensible",
		},
	}, {
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
		logs: []string{
			"level=debug msg=\"message received\" text=/start",
			"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=-1234",
		},
	}, {
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
		logs: []string{
			"level=debug msg=\"message received\" text=/stop",
			"level=info msg=\"user unsubscribed\" username=elliot user_id=123",
		},
	}, {
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
	}, {
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
		logs: []string{
			"level=debug msg=\"message received\" text=/start",
			"level=info msg=\"user subscribed\" username=elliot user_id=123 chat_id=123",
			"level=debug msg=\"message received\" text=/chats",
		},
	}, {
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
		logs: []string{
			"level=debug msg=\"message received\" text=/id",
		},
	}, {
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
	}, {
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
	}, {
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
)

func chatFromUser(user *telebot.User) *telebot.Chat {
	return &telebot.Chat{
		ID:        int64(user.ID),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Username:  user.Username,
		Type:      telebot.ChatPrivate,
	}
}

type testStore struct {
	// not thread safe - lol
	chats map[int64]*telebot.Chat
}

func (t *testStore) List() ([]*telebot.Chat, error) {
	chats := make([]*telebot.Chat, 0, len(t.chats))
	for _, chat := range t.chats {
		chats = append(chats, chat)
	}
	return chats, nil
}

func (t *testStore) Get(id telebot.ChatID) (*telebot.Chat, error) {
	chat, ok := t.chats[int64(id)]
	if !ok {
		return nil, telegram.ChatNotFoundErr
	}
	return chat, nil
}

func (t *testStore) Add(c *telebot.Chat) error {
	if t.chats == nil {
		t.chats = make(map[int64]*telebot.Chat)
	}
	t.chats[c.ID] = c
	return nil
}

func (t *testStore) Remove(_ *telebot.Chat) error {
	return nil
}

type reply struct {
	recipient, message string
}

// wraps telebot to intercept sent messages.
type testTelegram struct {
	bot     *telebot.Bot
	replies []reply
}

func (t *testTelegram) Start() {
	t.bot.Start()
}

func (t *testTelegram) Stop() {
	t.bot.Stop()
}

func (t *testTelegram) Send(to telebot.Recipient, message interface{}, _ ...interface{}) (*telebot.Message, error) {
	text, ok := message.(string)
	if !ok {
		return nil, fmt.Errorf("message is not a string")
	}
	t.replies = append(t.replies, reply{recipient: to.Recipient(), message: text})
	return nil, nil
}

func (t *testTelegram) Notify(_ telebot.Recipient, _ telebot.ChatAction) error {
	return nil // nop
}

func (t *testTelegram) Handle(endpoint interface{}, handler interface{}) {
	t.bot.Handle(endpoint, handler)
}

type testPoller struct {
	updates chan telebot.Update
	done    chan struct{}
}

func (t *testPoller) Poll(_ *telebot.Bot, updates chan telebot.Update, stop chan struct{}) {
	for {
		select {
		case upd := <-t.updates:
			updates <- upd
		case <-stop:
			return
		}
	}
}

func TestWorkflows(t *testing.T) {
	var testAlertmanagerAlerts func() string
	var testAlertmanagerStatus func() string
	var am *alertmanager.Client
	{
		m := http.NewServeMux()
		m.HandleFunc("/api/v2/alerts", func(w http.ResponseWriter, r *http.Request) {
			data := "[]"
			if testAlertmanagerAlerts != nil {
				data = testAlertmanagerAlerts()
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(data))
		})
		m.HandleFunc("/api/v2/status", func(w http.ResponseWriter, r *http.Request) {
			data := "{}"
			if testAlertmanagerStatus != nil {
				data = testAlertmanagerStatus()
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(data))
		})

		server := httptest.NewServer(m)
		defer server.Close()

		amURL, err := url.Parse(server.URL)
		require.NoError(t, err)
		am, err = alertmanager.NewClient(amURL)
		require.NoError(t, err)
	}

	for _, w := range workflows {
		t.Run(w.name, func(t *testing.T) {
			testAlertmanagerAlerts = w.alertmanagerAlerts
			testAlertmanagerStatus = w.alertmanagerStatus

			ctx, cancel := context.WithCancel(context.Background())
			logs := &bytes.Buffer{}

			poller := &testPoller{
				updates: make(chan telebot.Update, 2),
				done:    make(chan struct{}, 1),
			}
			tb, err := telebot.NewBot(telebot.Settings{
				Offline: true,
				Poller:  poller,
			})
			require.NoError(t, err)

			testStore := &testStore{}
			testTelegram := &testTelegram{bot: tb}

			bot, err := telegram.NewBotWithTelegram(testStore, testTelegram, admin.ID,
				telegram.WithLogger(log.NewLogfmtLogger(logs)),
				telegram.WithAlertmanager(am),
				telegram.WithTemplates(&url.URL{Host: "localhost"}, "../../default.tmpl"),
				telegram.WithStartTime(time.Now().Add(-time.Minute)),
				telegram.WithRevision("bot"),
			)
			require.NoError(t, err)

			webhooks := make(chan alertmanager.TelegramWebhook, 10)

			// Run the bot in the background and tests in foreground.
			go func(ctx context.Context) {
				require.NoError(t, bot.Run(ctx, webhooks))
			}(ctx)

			for i, update := range w.messages {
				update.ID = i
				update.Message.ID = i
				poller.updates <- update
				time.Sleep(time.Millisecond)
			}

			if w.webhooks != nil {
				for _, webhook := range w.webhooks() {
					webhooks <- webhook
				}
			}

			// TODO: Don't sleep but block somehow different
			time.Sleep(100 * time.Millisecond)

			require.Len(t, testTelegram.replies, len(w.replies))
			for i, reply := range w.replies {
				require.Equal(t, reply.recipient, testTelegram.replies[i].recipient)
				require.Equal(t, reply.message, strings.TrimSpace(testTelegram.replies[i].message))
			}

			logLines := strings.Split(strings.TrimSpace(logs.String()), "\n")

			require.Len(t, logLines, len(w.logs))
			for i, l := range w.logs {
				require.Equal(t, l, logLines[i])
			}

			cancel()
		})
	}
}
