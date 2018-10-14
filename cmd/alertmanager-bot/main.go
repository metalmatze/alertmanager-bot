package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/boltdb"
	"github.com/docker/libkv/store/consul"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/joho/godotenv"
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"github.com/oklog/run"
	"github.com/prometheus/alertmanager/template"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	storeBolt   = "bolt"
	storeConsul = "consul"

	levelDebug = "debug"
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
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
		alertmanager   *url.URL
		boltPath       string
		consul         *url.URL
		listenAddr     string
		logLevel       string
		logJSON        bool
		store          string
		telegramAdmins []int
		telegramToken  string
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

	a.Flag("log.json", "Tell the application to log json and not key value pairs").
		Envar("LOG_JSON").
		BoolVar(&config.logJSON)

	a.Flag("log.level", "The log level to use for filtering logs").
		Envar("LOG_LEVEL").
		Default(levelInfo).
		EnumVar(&config.logLevel, levelError, levelWarn, levelInfo, levelDebug)

	a.Flag("store", "The store to use").
		Required().
		Envar("STORE").
		EnumVar(&config.store, storeBolt, storeConsul)

	a.Flag("telegram.admin", "The ID of the initial Telegram Admin").
		Required().
		Envar("TELEGRAM_ADMIN").
		IntsVar(&config.telegramAdmins)

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

	levelFilter := map[string]level.Option{
		levelError: level.AllowError(),
		levelWarn:  level.AllowWarn(),
		levelInfo:  level.AllowInfo(),
		levelDebug: level.AllowDebug(),
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	if config.logJSON {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	}

	logger = level.NewFilter(logger, levelFilter[config.logLevel])
	logger = log.With(logger,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)

	tmpl, err := template.FromGlobs("default.tmpl")
	if err != nil {
		level.Error(logger).Log("msg", "failed to parse templates", "err", err)
		os.Exit(1)
	}
	tmpl.ExternalURL = config.alertmanager

	var kvStore store.Store
	{
		switch strings.ToLower(config.store) {
		case storeBolt:
			kvStore, err = boltdb.New([]string{config.boltPath}, &store.Config{Bucket: "alertmanager"})
			if err != nil {
				level.Error(logger).Log("msg", "failed to create bolt store backend", "err", err)
				os.Exit(1)
			}
		case storeConsul:
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

	ctx, cancel := context.WithCancel(context.Background())

	var g run.Group
	{
		tlogger := log.With(logger, "component", "telegram")

		chats, err := telegram.NewChatStore(kvStore)
		if err != nil {
			level.Error(logger).Log("msg", "failed to create chat store", "err", err)
			os.Exit(1)
		}

		bot, err := telegram.NewBot(
			chats, config.telegramToken, config.telegramAdmins[0],
			telegram.WithLogger(tlogger),
			telegram.WithAddr(config.listenAddr),
			telegram.WithAlertmanager(config.alertmanager),
			telegram.WithTemplates(tmpl),
			telegram.WithRevision(Revision),
			telegram.WithStartTime(StartTime),
			telegram.WithExtraAdmins(config.telegramAdmins[1:]...),
		)
		if err != nil {
			level.Error(tlogger).Log("msg", "failed to create bot", "err", err)
			os.Exit(2)
		}

		g.Add(func() error {
			level.Info(tlogger).Log(
				"msg", "starting alertmanager-bot",
				"version", Version,
				"revision", Revision,
				"buildDate", BuildDate,
				"goVersion", GoVersion,
			)

			// Runs the webserver in a goroutine sending incoming webhooks to Telegram
			go bot.RunWebserver()

			// Runs the bot itself communicating with Telegram
			bot.Run(ctx)
			return nil
		}, func(err error) {
			cancel()
		})
	}
	{
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, os.Kill)

		g.Add(func() error {
			<-sig
			return nil
		}, func(err error) {
			close(sig)
		})
	}

	if err := g.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
