package main

import (
	"fmt"
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

	responseStart = "Hey, %s! I will now keep you up to date!\n" + commandHelp
	responseStop  = "Alright, %s! I won't talk to you again.\n" + commandHelp
	responseHelp  = `
I'm a drone.io bot. I can notify you about your builds.

Available commands:
` + commandStart + ` - Start listening for drone.io builds
` + commandStop + `- Stop listening for drone.io builds
`
)

func main() {
	log.Println("starting...")
	bot, err := telebot.NewBot(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatalln(err)
	}

	messages := make(chan telebot.Message, 100)
	bot.Listen(messages, 1*time.Second)

	go HTTPListenAndServe()

	for message := range messages {
		switch message.Text {
		case commandStart:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
		case commandStop:
			bot.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
		case commandHelp:
			bot.SendMessage(message.Chat, responseHelp, nil)
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
