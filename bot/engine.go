package bot

// HandleFunc is used to generate the response to a request
type HandleFunc func(Context) error

// HandlerChain is the chain used for by command
type HandlerChain []HandleFunc

// Engine is the foundation for the bot
// Create a new one by using New()
type Engine struct {
	broker   []Broker
	commands map[string]HandlerChain
}

// New creates a new bot Engine
func New() (*Engine, error) {
	return &Engine{
		commands: make(map[string]HandlerChain),
	}, nil
}

// Run the telegram and listen to messages send to the telegram
func (e *Engine) Run() error {
	in := make(chan Context, 2014)
	for _, b := range e.broker {
		b.Run(in)
	}
	return nil
}

// AddBroker to the engine to communicate with
func (e *Engine) AddBroker(b Broker) {
	e.broker = append(e.broker, b)
}

// HandleFunc registers the handler function for the given command
func (e *Engine) HandleFunc(command string, handlers ...HandleFunc) {
	e.commands[command] = handlers
}
