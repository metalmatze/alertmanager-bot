package alertmanager

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/client_golang/prometheus"
)

type TelegramWebhook struct {
	ChatID  int64
	Message webhook.Message
}

// HandleTelegramWebhook returns a HandlerFunc that forwards webhooks to all bots via a channel.
func HandleTelegramWebhook(logger log.Logger, counter prometheus.Counter, webhooks chan<- TelegramWebhook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		chatID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/webhooks/telegram/"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unable to parse chat ID to int64"}`))
			return
		}

		var message webhook.Message

		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			level.Warn(logger).Log(
				"msg", "failed to decode webhook message",
				"err", err,
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		level.Debug(logger).Log(
			"msg", "received webhook",
			"alerts", len(message.Alerts),
			"chat_id", chatID,
		)

		webhooks <- TelegramWebhook{ChatID: chatID, Message: message}
		counter.Inc()
	}
}
