package alertmanager

import (
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

// HandleWebhook returns a HandlerFunc that sends messages for users via a channel
func HandleWebhook(logger log.Logger, counter prometheus.Counter, messages chan<- string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var webhook notify.WebhookMessage

		decoder := json.NewDecoder(r.Body)
		defer func() {
			if err := r.Body.Close(); err != nil {
				level.Warn(logger).Log(
					"msg", "can't close response body",
					"err", err,
				)
			}
		}()

		if err := decoder.Decode(&webhook); err != nil {
			level.Warn(logger).Log(
				"msg", "failed to decode webhook message",
				"err", err,
			)
		}

		for _, webAlert := range webhook.Alerts {
			labels := make(map[model.LabelName]model.LabelValue)
			for k, v := range webAlert.Labels {
				labels[model.LabelName(k)] = model.LabelValue(v)
			}

			annotations := make(map[model.LabelName]model.LabelValue)
			for k, v := range webAlert.Annotations {
				annotations[model.LabelName(k)] = model.LabelValue(v)
			}

			alert := &types.Alert{
				Alert: model.Alert{
					StartsAt:     webAlert.StartsAt,
					EndsAt:       webAlert.EndsAt,
					GeneratorURL: webAlert.GeneratorURL,
					Labels:       labels,
					Annotations:  annotations,
				},
			}

			var out string
			out = out + AlertMessage(alert) + "\n"

			messages <- out
		}

		counter.Inc()

		w.WriteHeader(http.StatusOK)
	}
}
