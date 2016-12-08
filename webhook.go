package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"github.com/tucnak/telebot"
)

// WebhookListen starts a http server and listens for incoming alerts to send to the users
func WebhookListen(addr string, bot *telebot.Bot, users *UserStore) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var webhook notify.WebhookMessage

		var buf bytes.Buffer
		tee := io.TeeReader(r.Body, &buf)
		defer r.Body.Close()

		decoder := json.NewDecoder(tee)
		if err := decoder.Decode(&webhook); err != nil {
			log.Printf("failed to decode webhook message: %v\n", err)
		}

		body, err := ioutil.ReadAll(&buf)
		if err != nil {
			log.Printf("failed to read from request.Body for logging: %v", err)
		}
		log.Println(string(body))

		for _, webAlert := range webhook.Alerts {
			labels := make(map[model.LabelName]model.LabelValue)
			for k, v := range webAlert.Labels {
				labels[model.LabelName(k)] = model.LabelValue(v)
			}

			annotations := make(map[model.LabelName]model.LabelValue)
			for k, v := range webAlert.Annotations {
				annotations[model.LabelName(k)] = model.LabelValue(v)
			}

			alert := types.Alert{
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

			for _, user := range users.List() {
				bot.SendMessage(user, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	if addr == "" {
		addr = ":8080"
	}

	log.Fatalln(http.ListenAndServe(addr, nil))
}
