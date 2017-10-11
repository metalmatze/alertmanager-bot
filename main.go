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
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/boltdb"
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

	// Create the config with default values
	config := Config{ListenAddr: ":8080"}

	if err := godotenv.Load(); err != nil {
		level.Info(logger).Log(
			"msg", "can't load .env",
			"err", err,
		)
	}
	arg.MustParse(&config)

	var users *UserStore
	{
		kvStore, err := boltdb.New([]string{config.Store}, &store.Config{Bucket: "alertmanager"})
		if err != nil {
			level.Error(logger).Log("msg", "failed to create store backend", "err", err)
			os.Exit(1)
		}
		defer kvStore.Close()

		users, err = NewUserStore(kvStore)
		if err != nil {
			level.Error(logger).Log("msg", "failed to create user store", "err", err)
			os.Exit(1)
		}
	}

	bot, err := NewBot(logger, config, users)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create bot", "err", err)
		os.Exit(2)
	}

	level.Info(logger).Log(
		"msg", "starting alertmanager-bot",
		"version", Version,
		"revision", Revision,
		"buildDate", BuildDate,
		"goVersion", GoVersion,
	)

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
