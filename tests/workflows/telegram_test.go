package workflows

import (
	"bytes"
	"context"
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
		name     string
		messages []telebot.Message
		replies  []testTelegramReply
		logs     []string
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
			Text:   "/incomprehensible",
			Chat: telebot.Chat{
				ID:        int64(admin.ID),
				FirstName: admin.FirstName,
				LastName:  admin.LastName,
				Username:  admin.Username,
				Type:      telebot.ChatPrivate,
			},
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
			Text:   "/start",
			Chat: telebot.Chat{
				ID:        int64(admin.ID),
				FirstName: admin.FirstName,
				LastName:  admin.LastName,
				Username:  admin.Username,
				Type:      telebot.ChatPrivate,
			},
		}},
		replies: []testTelegramReply{{
			recipient: "123",
			message:   "Hey, Elliot! I will now keep you up to date!\n/help",
		}},
		logs: []string{
			"level=debug msg=\"message received\" text=/start",
		},
	}}
)

type testStore struct {
	chats map[int64]telebot.Chat
}

func (t *testStore) List() ([]telebot.Chat, error) {
	chats := make([]telebot.Chat, len(t.chats))
	for i, chat := range t.chats {
		chats[i] = chat
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
	for _, w := range workflows {
		t.Run(w.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			logs := &bytes.Buffer{}

			testStore := &testStore{}
			testTelegram := &testTelegram{messages: w.messages}

			bot, err := telegram.NewBotWithTelegram(testStore, testTelegram, admin.ID,
				telegram.WithLogger(log.NewLogfmtLogger(logs)),
			)
			require.NoError(t, err)

			// Run the bot in the background and tests in foreground.
			go func(ctx context.Context) {
				err = bot.Run(ctx, make(chan notify.WebhookMessage))
				require.NoError(t, err)
			}(ctx)

			// TODO: Don't sleep but block somehow different
			time.Sleep(time.Second)

			require.Len(t, testTelegram.replies, len(w.replies))
			for i, reply := range w.replies {
				require.Equal(t, reply.recipient, testTelegram.replies[i].recipient)
				require.Equal(t, reply.message, testTelegram.replies[i].message)
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
