package main

import (
	"log"

	arg "github.com/alexflint/go-arg"
	"github.com/joho/godotenv"
)

var (
	// BuildTime is the time the binary was built
	BuildTime string
	// Commit is the git commit the binary was built from
	Commit string
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
