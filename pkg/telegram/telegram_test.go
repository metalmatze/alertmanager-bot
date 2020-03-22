package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/tucnak/telebot.v2"
)

func TestBot_handleStart(t *testing.T) {
	tm := &telebotMock{}
	b := &Bot{
		store:   NewChatStore(),
		telebot: tm,
	}

	tests := []struct {
		name    string
		message *telebot.Message
	}{
		{
			name: "simple",
			message: &telebot.Message{
				Chat: &telebot.Chat{
					ID: 1234,
				},
				Sender: &telebot.User{
					ID:        666,
					FirstName: "metalmatze",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm.testSend = func(to telebot.Recipient, what interface{}, options ...interface{}) {
				assert.Equal(t, &telebot.Chat{ID: 1234}, to)
				assert.Equal(t, "Hey, metalmatze! I will now keep you up to date!\n/help", what)
			}

			err := b.handleStart(tt.message)
			assert.NoError(t, err)
		})
	}
}
