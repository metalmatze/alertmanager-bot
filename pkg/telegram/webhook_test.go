package telegram

import (
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/assert"
	"gopkg.in/tucnak/telebot.v2"
)

type telebotMock struct {
	testSend func(to telebot.Recipient, what interface{}, options ...interface{})
}

func (t *telebotMock) Start() {
	panic("implement Start")
}

func (t *telebotMock) Stop() {
	panic("implement Stop")
}

func (t *telebotMock) Send(to telebot.Recipient, what interface{}, options ...interface{}) (*telebot.Message, error) {
	t.testSend(to, what, options)

	return &telebot.Message{}, nil
}

func TestBot_sendWebhook(t *testing.T) {
	tm := &telebotMock{}
	b := &Bot{
		store:   NewChatStore(),
		telebot: tm,
	}
	WithTemplate(&url.URL{}, "../../default.tmpl")(b)

	b.store.Add(&telebot.Chat{ID: 1234})

	tests := []struct {
		name        string
		message     webhook.Message
		messageText string
	}{
		{
			name: "simple",
			message: webhook.Message{
				Data: &template.Data{
					Receiver: "telegram",
					Status:   "firing",
					Alerts: template.Alerts{{
						Status:      "firing",
						Labels:      template.KV{"alertname": "Fire", "severity": "critical"},
						Annotations: template.KV{"message": "Something is on fire"},
						StartsAt:    time.Now().Add(-1 * time.Minute),
						EndsAt:      time.Now(),
					}},
					GroupLabels:       template.KV{"alertname": "Fire"},
					CommonLabels:      template.KV{"alertname": "Fire", "severity": "critical"},
					CommonAnnotations: template.KV{"message": "Something is on fire"},
				},
			},
			messageText: "\n\n🔥 <b>FIRING</b> 🔥\n<b>Fire</b>\nSomething is on fire\n<b>Duration:</b> 1 minute\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm.testSend = func(to telebot.Recipient, what interface{}, options ...interface{}) {
				assert.Equal(t, "1234", to.Recipient())
				assert.Equal(t, tt.messageText, what)
			}

			err := b.sendWebhook(tt.message)
			assert.NoError(t, err)
		})
	}
}
