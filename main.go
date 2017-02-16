package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/cenkalti/backoff"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/levels"
	"github.com/joho/godotenv"
	"github.com/metalmatze/alertmanager-bot/bot"
)

var (
	// BuildTime is the time the binary was built
	BuildTime string
	// Commit is the git commit the binary was built from
	Commit string
	// StartTime is the time the program was started
	StartTime time.Time
)

// Config knows all configurations from ENV
type Config struct {
	AlertmanagerURL string `arg:"env:ALERTMANAGER_URL"`
	TelegramToken   string `arg:"env:TELEGRAM_TOKEN"`
	TelegramAdmin   int    `arg:"env:TELEGRAM_ADMIN"`
	Store           string `arg:"env:STORE"`
	ListenAddr      string `arg:"env:LISTEN_ADDR"`
}

func main() {
	StartTime = time.Now()

	logWriter := log.NewSyncWriter(os.Stderr)
	logger := levels.New(log.NewLogfmtLogger(logWriter))

	if err := godotenv.Load(); err != nil {
		logger.Info().Log(
			"msg", "can't load .env",
			"err", err,
		)
	}

	logger.Debug().Log(
		"msg", "starting alertmanager-bot",
		"buildtime", BuildTime,
		"commit", Commit,
	)

	config := Config{
		ListenAddr: ":8080",
	}
	arg.MustParse(&config)

	alertmanagerBot, err := NewAlertmanagerBot(logger, config)
	if err != nil {
		logger.Error().Log(
			"msg", "failed to create alertmanager bot",
			"err", err,
		)
	}

	messages := make(chan string, 1024)
	go alertmanagerBot.RunWebserver(messages)

	b, err := bot.New()
	if err != nil {
		logger.Error().Log(
			"msg", "failed to create bot",
			"err", err,
		)
	}

	telegram, err := bot.NewTelegramBroker(b, config.TelegramToken)
	if err != nil {
		logger.Error().Log(
			"msg", "failed to create telegram broker",
			"err", err,
		)
	}
	b.AddBroker(telegram)

	b.Use(bot.Auth(config.TelegramAdmin), bot.Instrument(commandsCounter))

	b.HandleFunc(commandStart, alertmanagerBot.handleStart)
	b.HandleFunc(commandStop, alertmanagerBot.handleStop)
	b.HandleFunc(commandHelp, alertmanagerBot.handleHelp)
	b.HandleFunc(commandUsers, alertmanagerBot.handleUsers)
	b.HandleFunc(commandStatus, alertmanagerBot.handleStatus)
	b.HandleFunc(commandAlerts, alertmanagerBot.handleAlerts)
	b.HandleFunc(commandSilences, alertmanagerBot.handleSilences)

	if err := b.Run(); err != nil {
		logger.Error().Log(
			"msg", "failed to run bot",
			"err", err,
		)
	}
}

func httpGetBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 200 * time.Millisecond
	b.MaxInterval = 2 * time.Second
	b.MaxElapsedTime = 5 * time.Second // Telegram shows "typing" max 5 seconds
	return b
}

func httpGetRetry(logger levels.Levels, url string) (*http.Response, error) {
	var resp *http.Response
	var err error

	get := func() error {
		resp, err = http.Get(url)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("status code is %d not 200", resp.StatusCode)
		}

		return nil
	}

	notify := func(err error, dur time.Duration) {
		logger.Info().Log(
			"msg", "retrying",
			"duration", dur,
			"err", err,
			"url", url,
		)
	}

	if err := backoff.RetryNotify(get, httpGetBackoff(), notify); err != nil {
		return nil, err
	}

	return resp, err
}
