package bot

import (
	"fmt"

	"github.com/nlopes/slack"
	"github.com/tucnak/telebot"
)

// SlackBroker implements the Broker interface and
// allows communication between the bot and slack
type SlackBroker struct {
	engine      *Engine
	slackClient *slack.Client
	rtm         *slack.RTM
}

// NewSlackBroker returns a SlackBroker that's connected to slack
func NewSlackBroker(e *Engine, token string) (*SlackBroker, error) {
	return &SlackBroker{
		engine:      e,
		slackClient: slack.New(token),
	}, nil
}

// Name returns the name of the Broker: slack
func (b *SlackBroker) Name() string {
	return "slack"
}

// Run the SlackBroker and receive incoming messages via channel
func (b *SlackBroker) Run(done chan<- bool, in chan<- Context) { //TODO: Use in channel
	b.rtm = b.slackClient.NewRTM()
	go b.rtm.ManageConnection()

	for msg := range b.rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:

		case *slack.ConnectedEvent:
			fmt.Println("ConnectedEvent:", ev.ConnectionCount)
			b.rtm.SendMessage(b.rtm.NewOutgoingMessage("Hello world", "U17HBT4UR"))

		case *slack.MessageEvent:
			fmt.Println("MessageEvent:", ev.Text)

			ctx := &SlackContext{broker: b, message: ev}
			if handlers, ok := b.engine.commands[ev.Text]; ok {
				for _, handler := range handlers {
					if err := handler(ctx); err != nil {
						b.slackClient.PostMessage(
							ev.User,
							err.Error(),
							slack.NewPostMessageParameters(),
						)
					}
				}
			} else {
				for _, handler := range b.engine.notFound {
					if err := handler(ctx); err != nil {
						b.slackClient.PostMessage(
							ev.User,
							err.Error(),
							slack.NewPostMessageParameters(),
						)
					}
				}
			}

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return

		default:
			// Ignore other events..
			//fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}

	done <- true
}

// SlackContext implements the Context interface and
// makes sure everything is passed on to slack
type SlackContext struct {
	broker  *SlackBroker
	message *slack.MessageEvent
}

// Broker returns the name of the broker
func (c *SlackContext) Broker() string {
	return c.broker.Name()
}

// Raw returns the raw text of the incoming message
func (c *SlackContext) Raw() string {
	return c.message.Text
}

// User returns the user of the incoming message
func (c *SlackContext) User() telebot.User { // TODO: User
	return telebot.User{}
}

// Write

// String sends a string back to the user
func (c *SlackContext) String(msg string) error {
	user := c.message.User
	params := slack.NewPostMessageParameters()
	params.Username = "alertmanager"

	_, _, err := c.broker.slackClient.PostMessage(user, msg, params)
	return err
}

// Markdown sends a markdown formatted string back to the user
func (c *SlackContext) Markdown(msg string) error {
	panic("implement me")
}
