package alertmanager

import (
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/client_golang/prometheus"
)

// HandleWebhook returns a HandlerFunc that forwards webhooks to all bots via a channel
func HandleWebhook(logger log.Logger, counter prometheus.Counter, webhooks chan<- notify.WebhookMessage) http.HandlerFunc {
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

		var webhook notify.WebhookMessage

		err := json.NewDecoder(r.Body).Decode(&webhook)
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
			"alerts", len(webhook.Alerts),
		)

		webhooks <- webhook
		counter.Inc()
	}
}
