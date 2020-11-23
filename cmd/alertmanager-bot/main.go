package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/boltdb"
	"github.com/docker/libkv/store/consul"
	"github.com/docker/libkv/store/etcd"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hako/durafmt"
	"github.com/joho/godotenv"
	"github.com/metalmatze/alertmanager-bot/pkg/alertmanager"
	"github.com/metalmatze/alertmanager-bot/pkg/telegram"
	"github.com/oklog/run"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	storeBolt   = "bolt"
	storeConsul = "consul"
	storeEtcd   = "etcd"

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
		alertmanager           *url.URL
		boltPath               string
		consul                 *url.URL
		etcd                   *url.URL
		etcdInsecure           bool
		etcdInsecureSkipVerify bool
		etcdCertFile           string
		etcdKeyFile            string
		etcdCAFile             string
		listenAddr             string
		logLevel               string
		logJSON                bool
		store                  string
		telegramAdmins         []int
		telegramToken          string
		templatesPaths         []string
		storeKeyPrefix         string
	}{}

	a := kingpin.New("alertmanager-bot", "Bot for Prometheus' Alertmanager")
	a.HelpFlag.Short('h')

	a.Flag("alertmanager.url", "The URL that's used to connect to the alertmanager").
		Envar("ALERTMANAGER_URL").
		Default("http://localhost:9093/").
		URLVar(&config.alertmanager)

	a.Flag("bolt.path", "The path to the file where bolt persists its data").
		Envar("BOLT_PATH").
		Default("/tmp/bot.db").
		StringVar(&config.boltPath)

	a.Flag("consul.url", "The URL that's used to connect to the consul store").
		Envar("CONSUL_URL").
		Default("localhost:8500").
		URLVar(&config.consul)

	a.Flag("etcd.url", "The URL that's used to connect to the etcd store").
		Envar("ETCD_URL").
		Default("localhost:2379").
		URLVar(&config.etcd)

	a.Flag("etcd.tls.insecure", "Use TLS or not").
		Envar("ETCD_TLS_INSECURE").
		Default("false").
		BoolVar(&config.etcdInsecure)

	a.Flag("etcd.tls.insecureSkipVerify", "Skip server certificates verification").
		Envar("ETCD_TLS_INSECURE_SKIP_VERIFY").
		Default("false").
		BoolVar(&config.etcdInsecureSkipVerify)

	a.Flag("etcd.tls.cert", "Path to the TLS cert file").
		Envar("ETCD_TLS_CERT").
		StringVar(&config.etcdCertFile)

	a.Flag("etcd.tls.key", "Path to the TLS key file").
		Envar("ETCD_TLS_KEY").
		StringVar(&config.etcdKeyFile)

	a.Flag("etcd.tls.ca", "Path to the TLS trusted CA cert file").
		Envar("ETCD_TLS_CACERT").
		StringVar(&config.etcdCAFile)

	a.Flag("listen.addr", "The address the alertmanager-bot listens on for incoming webhooks").
		Envar("LISTEN_ADDR").
		Default("0.0.0.0:8080").
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
		EnumVar(&config.store, storeBolt, storeConsul, storeEtcd)

	a.Flag("storeKeyPrefix", "Prefix for store keys").
		Default("telegram/chats").
		Envar("STORE_KEY_PREFIX").
		StringVar(&config.storeKeyPrefix)

	a.Flag("telegram.admin", "The ID of the initial Telegram Admin").
		Required().
		Envar("TELEGRAM_ADMIN").
		IntsVar(&config.telegramAdmins)

	a.Flag("telegram.token", "The token used to connect with Telegram").
		Required().
		Envar("TELEGRAM_TOKEN").
		StringVar(&config.telegramToken)

	a.Flag("template.paths", "The paths to the template").
		Envar("TEMPLATE_PATHS").
		Default("/templates/default.tmpl").
		ExistingFilesVar(&config.templatesPaths)

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

	var tmpl *template.Template
	{
		funcs := template.DefaultFuncs
		funcs["since"] = func(t time.Time) string {
			return durafmt.Parse(time.Since(t)).String()
		}
		funcs["duration"] = func(start time.Time, end time.Time) string {
			return durafmt.Parse(end.Sub(start)).String()
		}

		template.DefaultFuncs = funcs

		tmpl, err = template.FromGlobs(config.templatesPaths...)
		if err != nil {
			level.Error(logger).Log("msg", "failed to parse templates", "err", err)
			os.Exit(1)
		}
		tmpl.ExternalURL = config.alertmanager
	}

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
		case storeEtcd:
			tlsConfig := &tls.Config{}

			if config.etcdCertFile != "" {
				cert, err := tls.LoadX509KeyPair(config.etcdCertFile, config.etcdKeyFile)
				if err != nil {
					level.Error(logger).Log("msg", "failed to create etcd store backend, could not load certificates", "err", err)
					os.Exit(1)
				}
				tlsConfig.Certificates = []tls.Certificate{cert}
			}

			if config.etcdCAFile != "" {
				caCert, err := ioutil.ReadFile(config.etcdCAFile)
				if err != nil {
					level.Error(logger).Log("msg", "failed to create etcd store backend, could not load ca certificate", "err", err)
					os.Exit(1)
				}

				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				tlsConfig.RootCAs = caCertPool
			}

			tlsConfig.InsecureSkipVerify = config.etcdInsecureSkipVerify

			if !config.etcdInsecure {
				kvStore, err = etcd.New([]string{config.etcd.String()}, &store.Config{TLS: tlsConfig})
			} else {
				kvStore, err = etcd.New([]string{config.etcd.String()}, nil)
			}

			if err != nil {
				level.Error(logger).Log("msg", "failed to create etcd store backend", "err", err)
				os.Exit(1)
			}
		default:
			level.Error(logger).Log("msg", "please provide one of the following supported store backends: bolt, consul, etcd")
			os.Exit(1)
		}
	}
	defer kvStore.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// TODO Needs fan out for multiple bots
	webhooks := make(chan notify.WebhookMessage, 32)

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

			// Runs the bot itself communicating with Telegram
			return bot.Run(ctx, webhooks)
		}, func(err error) {
			cancel()
		})
	}
	{
		wlogger := log.With(logger, "component", "webserver")

		// TODO: Use Heptio's healthcheck library
		handleHealth := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		webhooksCounter := prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "alertmanagerbot",
			Name:      "webhooks_total",
			Help:      "Number of webhooks received by this bot",
		})

		prometheus.MustRegister(webhooksCounter)

		m := http.NewServeMux()
		m.HandleFunc("/", alertmanager.HandleWebhook(wlogger, webhooksCounter, webhooks))
		m.Handle("/metrics", promhttp.Handler())
		m.HandleFunc("/health", handleHealth)
		m.HandleFunc("/healthz", handleHealth)

		s := http.Server{
			Addr:    config.listenAddr,
			Handler: m,
		}

		g.Add(func() error {
			level.Info(wlogger).Log("msg", "starting webserver", "addr", config.listenAddr)
			return s.ListenAndServe()
		}, func(err error) {
			s.Shutdown(context.Background())
		})
	}
	{
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, os.Kill)

		g.Add(func() error {
			<-sig
			return nil
		}, func(err error) {
			cancel()
			close(sig)
		})
	}

	if err := g.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
