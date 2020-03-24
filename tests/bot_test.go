package tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/tucnak/telebot.v2"
)

type alertmanagerMock struct{}

func (a alertmanagerMock) ListAlerts() ([]*types.Alert, error) {
	return []*types.Alert{{
		Alert: model.Alert{
			Labels:       model.LabelSet{"alertname": "Fire", "severity": "critical"},
			Annotations:  model.LabelSet{"message": "Something is on fire"},
			StartsAt:     time.Now().Add(-5 * time.Minute),
			EndsAt:       time.Now().Add(-1 * time.Minute),
			GeneratorURL: "",
		},
	}}, nil
}

type fakeBot struct {
	send    chan []byte
	receive chan []byte
}

func (fb fakeBot) sendText(text string) {
	payload := `{"ok": true, "result": [{"update_id": 451099383, "message": {"message_id": 1, "from": {"id": 1234567, "is_bot": false, "first_name": "Matthias", "last_name": "Loibl", "username": "MetalMatze", "language_code": "en"}, "chat": {"id": -419987192, "title": "AlertmanagerBotTests", "type": "group", "all_members_are_administrators": true}, "date": 1585000776, "text": "%s", "entities": [{"offset": 0, "length": 5, "type": "bot_command"}]}}]}`
	payload = fmt.Sprintf(payload, text)
	fb.send <- []byte(payload)
}

func (fb fakeBot) getText() (string, error) {
	var message struct {
		Text string `json:"text"`
	}

	payload := <-fb.receive

	err := json.Unmarshal(payload, &message)
	return message.Text, err
}

func TestBot(t *testing.T) {
	fb := fakeBot{
		send:    make(chan []byte, 32),
		receive: make(chan []byte, 32),
	}

	s := httptest.NewServer(api(t, fb))
	defer s.Close()

	tb, err := telebot.NewBot(telebot.Settings{
		URL:    s.URL,
		Poller: &telebot.LongPoller{Timeout: time.Second},
	})
	assert.NoError(t, err)

	opts := []telegram.BotOption{
		telegram.WithLogger(log.NewNopLogger()),
		telegram.WithTelebot(tb),
		telegram.WithTemplate(&url.URL{Host: "localhost"}, "../default.tmpl"),
	}
	bot, err := telegram.NewBot(telegram.NewChatStore(), alertmanagerMock{}, "test", opts...)
	assert.NoError(t, err)

	go bot.Run(make(chan webhook.Message))
	defer bot.Shutdown()

	type command struct {
		in  string
		out string
	}

	testcases := []struct {
		name     string
		commands []command
	}{
		{
			name: "Start",
			commands: []command{{
				in:  "/start",
				out: "Hey, Matthias! I will now keep you up to date!\n/help",
			}},
		},
		{
			name: "Stop",
			commands: []command{{
				in:  "/stop",
				out: "Alright, Matthias! I won't talk to you again.\n/help",
			}},
		},
		{
			name: "AlertsResolved",
			commands: []command{{
				in:  "/alerts",
				out: "\n\n<b>RESOLVED</b>\n<b>Fire</b>\nSomething is on fire\n<b>Duration:</b> 4 minutes\n<b>Ended:</b> 1 minute\n\n",
			}},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			for _, command := range tc.commands {
				fb.sendText(command.in)
				out, err := fb.getText()
				assert.NoError(t, err)
				assert.Equal(t, command.out, out)
			}
		})
	}
}

func api(t *testing.T, fb fakeBot) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.String() {
		case "/bot/getMe":
			resp := struct {
				Ok     bool         `json:"ok"`
				Result telebot.User `json:"result"`
			}{
				Ok: true,
				Result: telebot.User{
					ID:        1234,
					FirstName: "AlertmanagerTestBot",
					Username:  "AlertmanagerTestBot",
					IsBot:     true,
				},
			}

			payload, err := json.Marshal(resp)
			assert.NoError(t, err)
			_, _ = w.Write(payload)
		case "/bot/getUpdates":
			select {
			case update := <-fb.send:
				w.Write(update)
			default:
				w.Write([]byte(`{"ok":true,"result":[]}`))
			}
		case "/bot/sendMessage":
			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			defer r.Body.Close()

			fb.receive <- body
		default:
			panic(fmt.Sprintf("API mock missing endpoint %s", r.URL.String()))
		}
	}
}
