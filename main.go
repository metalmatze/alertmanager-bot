package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/cenkalti/backoff"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/joho/godotenv"
)

var (
	// Version of alertmanager-bot.
	Version string
	// Revision or Commit this binary was built from.
	Revision string
	// BuildDate this binary was built.
	BuildDate string
	// GoVersion running this binary.
	GoVersion = runtime.Version()
	// StartTime has the time this was started.
	StartTime = time.Now()
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
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, level.AllowAll())
	logger = log.With(logger,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)

	if err := godotenv.Load(); err != nil {
		level.Info(logger).Log(
			"msg", "can't load .env",
			"err", err,
		)
	}

	level.Info(logger).Log(
		"msg", "starting alertmanager-bot",
		"version", Version,
		"revision", Revision,
		"buildDate", BuildDate,
		"goVersion", GoVersion,
	)

	config := Config{
		ListenAddr: ":8080",
	}
	arg.MustParse(&config)

	bot, err := NewBot(logger, config)
	if err != nil {
		level.Debug(logger).Log("err", err)
	}

	go bot.RunWebserver()

	go bot.SendAdminMessage(
		config.TelegramAdmin,
		"alertmanager-bot just started. Please /start again to subscribe.",
	)

	bot.Run()
}

func httpGetBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 200 * time.Millisecond
	b.MaxInterval = 2 * time.Second
	b.MaxElapsedTime = 5 * time.Second // Telegram shows "typing" max 5 seconds
	return b
}

func httpGetRetry(logger log.Logger, url string) (*http.Response, error) {
	var resp *http.Response
	var err error

	get := func() error {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("status code is %d not 200", resp.StatusCode)
		}

		return nil
	}

	notify := func(err error, dur time.Duration) {
		level.Info(logger).Log(
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
