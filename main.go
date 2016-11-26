package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tucnak/telebot"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"

	responseStart = "Hey, %s! I will now keep you up to date! " + commandHelp
	responseStop  = "Alright, %s! I won't talk to you again. " + commandHelp
	responseHelp  = `
I'm a drone.io bot. I can notify you about your builds.

Available commands:
` + commandStart + ` - Start listening for drone.io builds
` + commandStop + `- Stop listening for drone.io builds
`
)

func main() {
	bot, err := telebot.NewBot(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatalln(err)
	}

	messages := make(chan telebot.Message, 100)
	bot.Listen(messages, 1*time.Second)

	go HTTPListenAndServe()

	for message := range messages {
		if message.Text == "/start" {
			bot.SendMessage(message.Chat, "Hello, "+message.Sender.FirstName+"!", nil)
		}
	}
}

func HTTPListenAndServe() {
	http.HandleFunc("/", Handle)

	log.Fatalln(http.ListenAndServe(":8080", nil))
}

func Handle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%s", b)

	w.WriteHeader(http.StatusOK)
}
