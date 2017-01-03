package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/cenkalti/backoff"
	"github.com/joho/godotenv"
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
	log.Println("starting alertmanager-telegram")
	log.Printf("BuildTime: %s, Commit: %s\n", BuildTime, Commit)
	StartTime = time.Now()

	if err := godotenv.Load(); err != nil {
		log.Println(err)
	}

	var c Config
	arg.MustParse(&c)

	bot, err := NewBot(c)
	if err != nil {
		log.Fatalln(err)
	}

	go bot.RunWebhook()

	bot.Run()
}

func httpGetBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 200 * time.Millisecond
	b.MaxInterval = 2 * time.Second
	b.MaxElapsedTime = 5 * time.Second // Telegram shows "typing" max 5 seconds
	return b
}

func httpGetRetry(url string) (*http.Response, error) {
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
		log.Printf("retrying in %v: %v", dur, err)
	}

	if err := backoff.RetryNotify(get, httpGetBackoff(), notify); err != nil {
		return nil, err
	}

	return resp, err
}
