package bot

// HandleFunc is used to generate the response to a request
type HandleFunc func(Context) error

// HandlerChain is the chain used for by command
type HandlerChain []HandleFunc

// Engine is the foundation for the bot
// Create a new one by using New()
type Engine struct {
	broker      []Broker
	middlewares HandlerChain
	commands    map[string]HandlerChain
	notFound    HandlerChain
}

var (
	// DefaultNotFoundHandler is the default response send to the user
	DefaultNotFoundHandler = func(c Context) error {
		return c.String("Sorry, I don't understand...")
	}
)

// New creates a new bot Engine
func New() (*Engine, error) {
	return &Engine{
		commands: make(map[string]HandlerChain),
		notFound: []HandleFunc{DefaultNotFoundHandler},
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

// Use adds middlewares to the engine that are run before every handler
func (e *Engine) Use(middlewares ...HandleFunc) {
	e.middlewares = append(e.middlewares, middlewares...)
}

// HandleFunc registers the handler function for the given command
func (e *Engine) HandleFunc(command string, handlers ...HandleFunc) {
	e.commands[command] = handlers
}

// HandleNotFound uses these exact handlers to create a response if no handler was found
func (e *Engine) HandleNotFound(handlers ...HandleFunc) {
	e.notFound = append(HandlerChain{}, handlers...)
}
