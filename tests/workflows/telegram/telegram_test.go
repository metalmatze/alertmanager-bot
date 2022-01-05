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

type workflow struct {
	name     string
	messages []telebot.Update
	replies  []reply
	logs     []string
	counter  map[string]uint

	webhooks           func() []alertmanager.TelegramWebhook
	alertmanagerAlerts func(t *testing.T, r *http.Request) string
	alertmanagerStatus func(t *testing.T, r *http.Request) string
}

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
	// The admin is determined by ID, so we can use different user objects with the same ID in the tests without creating inconsistent state.
	anonymousAdmin = &telebot.User{
		ID:    123,
		IsBot: false,
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
	workflows = []workflow{{
		name: "Dropped",
		messages: []telebot.Update{{
			Message: &telebot.Message{
				Sender: nobody,
			},
		}},
		replies: []reply{},
		counter: map[string]uint{},
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

type reply struct {
	recipient, message string
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

type testCommandCounter struct {
	counter map[string]uint
}

func (c *testCommandCounter) Count(command string) {
	c.counter[command]++
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
	var testAlertmanagerAlerts func(t *testing.T, r *http.Request) string
	var testAlertmanagerStatus func(t *testing.T, r *http.Request) string
	var am *alertmanager.Client
	{
		m := http.NewServeMux()
		m.HandleFunc("/api/v2/alerts", func(w http.ResponseWriter, r *http.Request) {
			data := "[]"
			if testAlertmanagerAlerts != nil {
				data = testAlertmanagerAlerts(t, r)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(data))
		})
		m.HandleFunc("/api/v2/status", func(w http.ResponseWriter, r *http.Request) {
			data := "{}"
			if testAlertmanagerStatus != nil {
				data = testAlertmanagerStatus(t, r)
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

	workflows = append(workflows, alertsWorkflows...)
	workflows = append(workflows, chatsWorkflows...)
	workflows = append(workflows, helpWorkflows...)
	workflows = append(workflows, idWorkflows...)
	workflows = append(workflows, startWorkflows...)
	workflows = append(workflows, stopWorkflows...)
	workflows = append(workflows, statusWorkflows...)
	workflows = append(workflows, webhookWorkflows...)

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
			counter := testCommandCounter{counter: map[string]uint{}}

			bot, err := telegram.NewBotWithTelegram(testStore, testTelegram, admin.ID,
				telegram.WithLogger(log.NewLogfmtLogger(logs)),
				telegram.WithCommandEvent(counter.Count),
				telegram.WithAlertmanager(am),
				telegram.WithTemplates(&url.URL{Host: "localhost"}, "../../../default.tmpl"),
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

			require.Len(t, counter.counter, len(w.counter))
			for command, count := range counter.counter {
				require.Equal(t, w.counter[command], count)
			}

			cancel()
		})
	}
}
