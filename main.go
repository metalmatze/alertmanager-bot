package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/boltdb"
	"github.com/docker/libkv/store/consul"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/joho/godotenv"
	"gopkg.in/alecthomas/kingpin.v2"
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

func main() {
	godotenv.Load()

	config := struct {
		alertmanager  *url.URL
		boltPath      string
		consul        *url.URL
		listenAddr    string
		store         string
		telegramAdmin int
		telegramToken string
	}{}

	a := kingpin.New("alertmanager-bot", "Bot for Prometheus' Alertmanager")
	a.HelpFlag.Short('h')

	a.Flag("alertmanager.url", "The URL that's used to connect to the alertmanager").
		Required().
		Envar("ALERTMANAGER_URL").
		URLVar(&config.alertmanager)

	a.Flag("bolt.path", "The path to the file where bolt persists its data").
		Envar("BOLT_PATH").
		StringVar(&config.boltPath)

	a.Flag("consul.url", "The URL that's used to connect to the consul store").
		Envar("CONSUL_URL").
		URLVar(&config.consul)

	a.Flag("listen.addr", "The address the alertmanager-bot listens on for incoming webhooks").
		Required().
		Envar("LISTEN_ADDR").
		StringVar(&config.listenAddr)

	a.Flag("store", "The store to use").
		Required().
		Envar("STORE").
		StringVar(&config.store)

	a.Flag("telegram.admin", "The ID of the initial Telegram Admin").
		Required().
		Envar("TELEGRAM_ADMIN").
		IntVar(&config.telegramAdmin)

	a.Flag("telegram.token", "The token used to connect with Telegram").
		Required().
		Envar("TELEGRAM_TOKEN").
		StringVar(&config.telegramToken)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("error parsing commandline arguments: %v\n", err)
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, level.AllowAll())
	logger = log.With(logger,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)

	var kvStore store.Store
	{
		switch strings.ToLower(config.store) {
		case "bolt":
			kvStore, err = boltdb.New([]string{config.boltPath}, &store.Config{Bucket: "alertmanager"})
			if err != nil {
				level.Error(logger).Log("msg", "failed to create bolt store backend", "err", err)
				os.Exit(1)
			}
		case "consul":
			kvStore, err = consul.New([]string{config.consul.String()}, nil)
			if err != nil {
				level.Error(logger).Log("msg", "failed to create consul store backend", "err", err)
				os.Exit(1)
			}
		default:
			level.Error(logger).Log("msg", "please provide one of the following supported store backends: bolt, consul")
			os.Exit(1)
		}
	}
	defer kvStore.Close()

	chats, err := NewChatStore(kvStore)
	if err != nil {
		level.Error(logger).Log("msg", "failed to create chat store", "err", err)
		os.Exit(1)
	}

	bot, err := NewBot(
		chats, config.telegramToken, config.telegramAdmin,
		BotWithLogger(logger),
		BotWithAddr(config.listenAddr),
		BotWithAlertmanager(config.alertmanager),
	)
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

	// Runs the webserver in a goroutine sending incoming webhooks to Telegram
	go bot.RunWebserver()

	// Runs the bot itself communicating with Telegram
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
