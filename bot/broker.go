package bot

// Broker is the glue between the bot and every messenger like e.g. Telegram or Slack
type Broker interface {
	Name() string
	Run(chan<- bool, chan<- Context)
}
