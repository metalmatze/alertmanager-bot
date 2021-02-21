package workflows

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
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"github.com/prometheus/alertmanager/notify"
	"github.com/stretchr/testify/require"
	"github.com/tucnak/telebot"
)

var (
	admin = telebot.User{
		ID:        123,
		FirstName: "Elliot",
		LastName:  "Alderson",
		Username:  "elliot",
		IsBot:     false,
	}
	nobody = telebot.User{
		ID:        222,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "nobody",
		IsBot:     false,
	}

	// These are the different workflows/scenarios we are testing.
	workflows = []struct {
		name               string
		messages           []telebot.Message
		replies            []testTelegramReply
		logs               []string
		alertmanagerAlerts func() string
		alertmanagerStatus func() string
	}{{
		name: "Dropped",
		messages: []telebot.Message{{
			Sender: nobody,
		}},
		replies: []testTelegramReply{},
		logs: []string{
			"level=info msg=\"failed to process message\" err=\"dropped message from forbidden sender\" sender_id=222 sender_username=nobody",
		},
	}, {
		name: "Incomprehensible",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   "/incomprehensible",
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "Sorry, I don't understand...",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/incomprehensible",
		},
	}, {
		name: "Start",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStart,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "Hey, Elliot! I will now keep you up to date!\n/help",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/start",
			"level=info msg=\"user subscribed\" username=elliot user_id=123",
		},
	}, {
		name: "StopWithoutStart",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStop,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "Alright, Elliot! I won't talk to you again.\n/help",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/stop",
			"level=info msg=\"user unsubscribed\" username=elliot user_id=123",
		},
	}, {
		name: "Help",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandHelp,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   strings.TrimSpace(telegram.ResponseHelp),
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/help",
		},
	}, {
		name: "HelpAsNobody",
		messages: []telebot.Message{{
			Sender: nobody,
			Chat:   chatFromUser(nobody),
			Text:   telegram.CommandHelp,
		}},
		replies: []testTelegramReply{},
		logs: []string{
			"level=info msg=\"failed to process message\" err=\"dropped message from forbidden sender\" sender_id=222 sender_username=nobody",
		},
	}, {
		name: "ChatsNone",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandChats,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "Currently no one is subscribed.",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/chats",
		},
	}, {
		name: "ChatsWithAdminSubscribed",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStart,
		}, {
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandChats,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "Hey, Elliot! I will now keep you up to date!\n/help",
		}, {
			recipient: "123",
			message:   "Currently these chat have subscribed:\n@elliot",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/start",
			"level=info msg=\"user subscribed\" username=elliot user_id=123",
			"level=debug msg=\"message received\" text=/chats",
		},
	}, {
		name: "IDAsNobody",
		messages: []telebot.Message{{
			Sender: nobody,
			Chat:   chatFromUser(nobody),
			Text:   telegram.CommandID,
		}},
		replies: []testTelegramReply{{
			recipient: "222",
			message:   "Your Telegram ID is 222",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/id",
		},
	}, {
		name: "IDAsAdmin",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandID,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "Your Telegram ID is 123",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/id",
		},
	}, {
		name: "Status",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandStatus,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "*AlertManager*\nVersion: alertmanager\nUptime: 1 minute\n*AlertManager Bot*\nVersion: bot\nUptime: 1 minute",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/status",
		},
		alertmanagerStatus: func() string {
			return fmt.Sprintf(
				`{"data":{"uptime":%q,"versionInfo":{"version":"alertmanager"}}}"`,
				time.Now().Add(-time.Minute).Format(time.RFC3339),
			)
		},
	}, {
		name: "AlertsNone",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "No alerts right now! 🎉",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/alerts",
		},
	}, {
		name: "AlertsFiring",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "🔥 <b>damn</b> 🔥\n<b>Labels:</b>\n    bot: alertmanager-bot\n<b>Annotations:</b>\n    msg: sup?!\n<b>Duration:</b> 1 hour",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/alerts",
		},
		alertmanagerAlerts: func() string {
			return fmt.Sprintf(
				`{"status":"success", "data":[{"labels":{"alertname":"damn","bot":"alertmanager-bot"},"annotations":{"msg":"sup?!"},"startsAt":%q}]}`,
				time.Now().Add(-time.Hour).Format(time.RFC3339),
			)
		},
	}, {
		name: "AlertsResolved",
		messages: []telebot.Message{{
			Sender: admin,
			Chat:   chatFromUser(admin),
			Text:   telegram.CommandAlerts,
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "✅ <b>damn</b> ✅\n<b>Labels:</b>\n    bot: alertmanager-bot\n<b>Annotations:</b>\n    msg: sup?!\n<b>Duration:</b> 58 minutes\n<b>Ended:</b> 2 minutes",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/alerts",
		},
		alertmanagerAlerts: func() string {
			return fmt.Sprintf(
				`{"status":"success", "data":[{"labels":{"alertname":"damn","bot":"alertmanager-bot"},"annotations":{"msg":"sup?!"},"startsAt":%q,"endsAt":%q}]}`,
				time.Now().Add(-time.Hour).Format(time.RFC3339),
				time.Now().Add(-2*time.Minute).Format(time.RFC3339),
			)
		},
	}}
)

func chatFromUser(user telebot.User) telebot.Chat {
	return telebot.Chat{
		ID:        int64(user.ID),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Username:  user.Username,
		Type:      telebot.ChatPrivate,
	}
}

type testStore struct {
	// not thread safe - lol
	chats map[int64]telebot.Chat
}

func (t *testStore) List() ([]telebot.Chat, error) {
	chats := make([]telebot.Chat, 0, len(t.chats))
	for _, chat := range t.chats {
		chats = append(chats, chat)
	}
	return chats, nil
}

func (t *testStore) Add(c telebot.Chat) error {
	if t.chats == nil {
		t.chats = make(map[int64]telebot.Chat)
	}
	t.chats[c.ID] = c
	return nil
}

func (t *testStore) Remove(_ telebot.Chat) error {
	return nil
}

type testTelegramReply struct {
	recipient, message string
}

type testTelegram struct {
	messages []telebot.Message
	replies  []testTelegramReply
}

func (t *testTelegram) Listen(messages chan telebot.Message, _ time.Duration) {
	for i, m := range t.messages {
		m.ID = i
		messages <- m
	}
}

func (t *testTelegram) SendChatAction(_ telebot.Recipient, _ telebot.ChatAction) error {
	return nil
}

func (t *testTelegram) SendMessage(recipient telebot.Recipient, message string, _ *telebot.SendOptions) error {
	t.replies = append(t.replies, testTelegramReply{recipient: recipient.Destination(), message: message})
	return nil
}

func TestWorkflows(t *testing.T) {
	var testAlertmanagerAlerts func() string
	var testAlertmanagerStatus func() string
	var testAlertmanagerURL *url.URL
	{
		m := http.NewServeMux()
		m.HandleFunc("/api/v1/alerts", func(w http.ResponseWriter, r *http.Request) {
			data := "{}"
			if testAlertmanagerAlerts != nil {
				data = testAlertmanagerAlerts()
			}
			_, _ = w.Write([]byte(data))
		})
		m.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
			data := "{}"
			if testAlertmanagerStatus != nil {
				data = testAlertmanagerStatus()
			}
			_, _ = w.Write([]byte(data))
		})

		server := httptest.NewServer(m)
		defer server.Close()

		testAlertmanagerURL, _ = url.Parse(server.URL)
	}

	for _, w := range workflows {
		t.Run(w.name, func(t *testing.T) {
			testAlertmanagerAlerts = w.alertmanagerAlerts
			testAlertmanagerStatus = w.alertmanagerStatus

			ctx, cancel := context.WithCancel(context.Background())
			logs := &bytes.Buffer{}

			testStore := &testStore{}
			testTelegram := &testTelegram{messages: w.messages}

			bot, err := telegram.NewBotWithTelegram(testStore, testTelegram, admin.ID,
				telegram.WithLogger(log.NewLogfmtLogger(logs)),
				telegram.WithAlertmanager(testAlertmanagerURL),
				telegram.WithTemplates(&url.URL{Host: "localhost"}, "../../default.tmpl"),
				telegram.WithStartTime(time.Now().Add(-time.Minute)),
				telegram.WithRevision("bot"),
			)
			require.NoError(t, err)

			// Run the bot in the background and tests in foreground.
			go func(ctx context.Context) {
				require.NoError(t, bot.Run(ctx, make(chan notify.WebhookMessage)))
			}(ctx)

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
