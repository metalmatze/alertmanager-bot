package alertmanager

import (
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/client_golang/prometheus"
)

// HandleWebhook returns a HandlerFunc that forwards webhooks to all bots via a channel
func HandleWebhook(logger log.Logger, counter prometheus.Counter, messages chan<- webhook.Message) http.HandlerFunc {
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

		var message webhook.Message

		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
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
		)

		messages <- message
		counter.Inc()
	}
}
