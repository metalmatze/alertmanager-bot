package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/hako/durafmt"
	"github.com/joho/godotenv"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/common/model"
	"github.com/tucnak/telebot"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"
	commandUsers = "/users"

	commandStatus     = "/status"
	commandAlerts     = "/alerts"
	commandSilences   = "/silences"
	commandSilenceAdd = "/silence_add"
	commandSilence    = "/silence"
	commandSilenceDel = "/silence_del"

	responseStart = "Hey, %s! I will now keep you up to date!\n" + commandHelp
	responseStop  = "Alright, %s! I won't talk to you again.\n" + commandHelp
	responseHelp  = `
I'm a Prometheus AlertManager bot for Telegram. I will notify you about alerts.
You can also ask me about my ` + commandStatus + `, ` + commandAlerts + ` & ` + commandSilences + `

Available commands:
` + commandStart + ` - Subscribe for alerts.
` + commandStop + ` - Unsubscribe for alerts.
` + commandStatus + ` - Print the current status.
` + commandAlerts + ` - List all alerts.
` + commandSilences + ` - List all silences.
`
)

var users map[int]telebot.User

type Config struct {
	AlertmanagerURL string `arg:"env:ALERTMANAGER_URL"`
	TelegramToken   string `arg:"env:TELEGRAM_TOKEN"`
	TelegramAdmin   int    `arg:"env:TELEGRAM_ADMIN"`
}

func main() {
	log.Println("starting...")

	if err := godotenv.Load(); err != nil {
		log.Println(err)
	}

	var c Config
	arg.MustParse(&c)

	bot, err := telebot.NewBot(c.TelegramToken)
	if err != nil {
		log.Fatalln(err)
	}

	users = make(map[int]telebot.User)
	messages := make(chan telebot.Message, 100)
	bot.Listen(messages, 1*time.Second)

	go HTTPListenAndServe(bot)

	for message := range messages {
		if message.Sender.ID != c.TelegramAdmin {
			log.Printf("dropped message from unallowed sender: %s(%d)", message.Sender.Username, message.Sender.ID)
			continue
		}

		switch message.Text {
		case commandStart:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
			users[message.Sender.ID] = message.Sender
			log.Printf("User %s(%d) subscribed", message.Sender.Username, message.Sender.ID)
		case commandStop:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
			delete(users, message.Sender.ID)
			log.Printf("User %s(%d) unsubscribed", message.Sender.Username, message.Sender.ID)
		case commandHelp:
			bot.SendMessage(message.Chat, responseHelp, nil)
		case commandUsers:
			bot.SendMessage(message.Chat, fmt.Sprintf("Currently %d users are subscribed.", len(users)), nil)
		case commandStatus:
			s, err := status(c)
			if err != nil {
				bot.SendMessage(message.Chat, fmt.Sprintf("failed to get status... %v", err), nil)
			}

			uptime := durafmt.Parse(time.Since(s.Data.Uptime))

			bot.SendMessage(
				message.Chat,
				fmt.Sprintf("Version: %s\nUptime: %s", s.Data.VersionInfo.Version, uptime),
				nil,
			)
		case commandAlerts:
			alerts, err := listAlerts(c)
			if err != nil {
				bot.SendMessage(message.Chat, fmt.Sprintf("failed to list alerts... %v", err), nil)
			}

			if len(alerts) == 0 {
				bot.SendMessage(message.Chat, "No alerts right now! 🎉", nil)
			}

			var out string
			for _, a := range alerts {
				out = out + Message(a) + "\n"
			}

			bot.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		}
	}
}

func HTTPListenAndServe(bot *telebot.Bot) {
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

		for _, alert := range webhook.Alerts {
			var out string
			out = out + Message(alert) + "\n"

			for _, user := range users {
				bot.SendMessage(user, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
			}

		}

		w.WriteHeader(http.StatusOK)
	})

	log.Fatalln(http.ListenAndServe(":8080", nil))
}

func Message(a template.Alert) string {
	if a.Status == "" {
		if a.EndsAt.IsZero() {
			a.Status = string(model.AlertFiring)
		} else {
			a.Status = string(model.AlertResolved)
		}
	}

	status := a.Status
	switch status {
	case string(model.AlertFiring):
		status = "🔥 *" + strings.ToUpper(status) + "* 🔥"
	case string(model.AlertResolved):
		status = "*" + strings.ToUpper(status) + "*"
	}

	return fmt.Sprintf(
		"%s\n*%s* (%s)\n%s\n",
		status,
		a.Labels["alertname"],
		a.Annotations["summary"],
		a.Annotations["description"],
	)
}

type alertResponse struct {
	Status string           `json:"status"`
	Alerts []template.Alert `json:"data,omitempty"`
}

func listAlerts(c Config) ([]template.Alert, error) {
	resp, err := http.Get(c.AlertmanagerURL + "/api/v1/alerts")
	if err != nil {
		return nil, err
	}

	var alertResponse alertResponse
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	dec.Decode(&alertResponse)

	return alertResponse.Alerts, err
}

type statusResponse struct {
	Status string `json:"status"`
	Data   struct {
		Uptime      time.Time `json:"uptime"`
		VersionInfo struct {
			Branch    string `json:"branch"`
			BuildDate string `json:"buildDate"`
			BuildUser string `json:"buildUser"`
			GoVersion string `json:"goVersion"`
			Revision  string `json:"revision"`
			Version   string `json:"version"`
		} `json:"versionInfo"`
	} `json:"data"`
}

func status(c Config) (statusResponse, error) {
	var statusResponse statusResponse

	resp, err := http.Get(c.AlertmanagerURL + "/api/v1/status")
	if err != nil {
		return statusResponse, err
	}

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	dec.Decode(&statusResponse)

	return statusResponse, nil
}
