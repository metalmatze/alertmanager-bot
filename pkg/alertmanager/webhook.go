package alertmanager

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/alertmanager/notify/webhook"
)

func HandleWebhook(webhooks chan<- webhook.Message) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		var m webhook.Message
		err := json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			return
		}

		webhooks <- m
	}
}
