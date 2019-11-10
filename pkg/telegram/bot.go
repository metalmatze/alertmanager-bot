package telegram

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-openapi/strfmt"
	"github.com/hako/durafmt"
	"github.com/metalmatze/alertmanager-bot/pkg/alertmanager"
	"github.com/oklog/run"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/tucnak/telebot"
)

const (
	commandStart = "/start"
	commandStop  = "/stop"
	commandHelp  = "/help"
	commandChats = "/chats"

	commandStatus     = "/status"
	commandAlerts     = "/alerts"
	commandSilences   = "/silences"
	commandSilenceAdd = "/silence_add"
	commandSilence    = "/silence"
	commandSilenceDel = "/silence_del"

	responseStart = "Hey, %s! I will now keep you up to date!\n" + commandHelp
	responseStop  = "Alright, %s! I won't talk to you again.\n" + commandHelp
	responseHelp  = `
I'm a Prometheus AlertManager Bot for Telegram. I will notify you about alerts.
You can also ask me about my ` + commandStatus + `, ` + commandAlerts + ` & ` + commandSilences + `

Available commands:
` + commandStart + ` - Subscribe for alerts.
` + commandStop + ` - Unsubscribe for alerts.
` + commandStatus + ` - Print the current status.
` + commandAlerts + ` - List all alerts.
` + commandSilences + ` - List all silences.
` + commandChats + ` - List all users and group chats that subscribed.
`
)

// BotChatStore is all the Bot needs to store and read
type BotChatStore interface {
	List() ([]telebot.Chat, error)
	Add(telebot.Chat) error
	Remove(telebot.Chat) error
}

// Bot runs the alertmanager telegram
type Bot struct {
	addr         string
	admins       []int // must be kept sorted
	templates    *template.Template
	alertmanager *client.Alertmanager
	chats        BotChatStore
	logger       log.Logger
	revision     string
	startTime    time.Time

	telegram *telebot.Bot

	commandsCounter *prometheus.CounterVec
	webhooksCounter prometheus.Counter
}

// BotOption passed to NewBot to change the default instance
type BotOption func(b *Bot)

// NewBot creates a Bot with the UserStore and telegram telegram
func NewBot(chats BotChatStore, token string, admin int, opts ...BotOption) (*Bot, error) {
	bot, err := telebot.NewBot(token)
	if err != nil {
		return nil, err
	}

	commandsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alertmanagerbot",
		Name:      "commands_total",
		Help:      "Number of commands received by command name",
	}, []string{"command"})
	if err := prometheus.Register(commandsCounter); err != nil {
		return nil, err
	}

	transport := client.DefaultTransportConfig().
		WithHost("localhost:9093")

	b := &Bot{
		logger:          log.NewNopLogger(),
		telegram:        bot,
		chats:           chats,
		addr:            "127.0.0.1:8080",
		admins:          []int{admin},
		alertmanager:    client.NewHTTPClientWithConfig(strfmt.Default, transport),
		commandsCounter: commandsCounter,
		// TODO: initialize templates with default?
	}

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

// WithLogger sets the logger for the Bot as an option
func WithLogger(l log.Logger) BotOption {
	return func(b *Bot) {
		b.logger = l
	}
}

// WithAddr sets the internal listening addr of the bot's web server receiving webhooks
func WithAddr(addr string) BotOption {
	return func(b *Bot) {
		b.addr = addr
	}
}

// WithAlertmanager sets the connection url for the Alertmanager
func WithAlertmanager(u *url.URL) BotOption {
	return func(b *Bot) {
		transport := client.DefaultTransportConfig().
			WithHost(u.Host).
			WithBasePath(u.Path)

		b.alertmanager = client.NewHTTPClientWithConfig(strfmt.Default, transport)
	}
}

// WithTemplates uses Alertmanager template to render messages for Telegram
func WithTemplates(t *template.Template) BotOption {
	return func(b *Bot) {
		b.templates = t
	}
}

// WithRevision is setting the Bot's revision for status commands
func WithRevision(r string) BotOption {
	return func(b *Bot) {
		b.revision = r
	}
}

// WithStartTime is setting the Bot's start time for status commands
func WithStartTime(st time.Time) BotOption {
	return func(b *Bot) {
		b.startTime = st
	}
}

// WithExtraAdmins allows the specified additional user IDs to issue admin
// commands to the bot.
func WithExtraAdmins(ids ...int) BotOption {
	return func(b *Bot) {
		b.admins = append(b.admins, ids...)
		sort.Ints(b.admins)
	}
}

// SendAdminMessage to the admin's ID with a message
func (b *Bot) SendAdminMessage(adminID int, message string) {
	b.telegram.SendMessage(telebot.User{ID: adminID}, message, nil)
}

// isAdminID returns whether id is one of the configured admin IDs.
func (b *Bot) isAdminID(id int) bool {
	i := sort.SearchInts(b.admins, id)
	return i < len(b.admins) && b.admins[i] == id
}

// Run the telegram and listen to messages send to the telegram
func (b *Bot) Run(ctx context.Context) error {
	commandSuffix := fmt.Sprintf("@%s", b.telegram.Identity.Username)

	commands := map[string]func(message telebot.Message){
		commandStart:    b.handleStart,
		commandStop:     b.handleStop,
		commandHelp:     b.handleHelp,
		commandChats:    b.handleChats,
		commandStatus:   b.handleStatus,
		commandAlerts:   b.handleAlerts,
		commandSilences: b.handleSilences,
	}

	// init counters with 0
	for command := range commands {
		b.commandsCounter.WithLabelValues(command).Add(0)
	}

	process := func(message telebot.Message) error {
		if message.IsService() {
			return nil
		}

		if !b.isAdminID(message.Sender.ID) {
			b.commandsCounter.WithLabelValues("dropped").Inc()
			return fmt.Errorf("dropped message from forbidden sender")
		}

		if err := b.telegram.SendChatAction(message.Chat, telebot.Typing); err != nil {
			return err
		}

		// Remove the command suffix from the text, /help@BotName => /help
		text := strings.Replace(message.Text, commandSuffix, "", -1)
		// Only take the first part into account, /help foo => /help
		text = strings.Split(text, " ")[0]

		level.Debug(b.logger).Log("msg", "message received", "text", text)

		// Get the corresponding handler from the map by the commands text
		handler, ok := commands[text]

		if !ok {
			b.commandsCounter.WithLabelValues("incomprehensible").Inc()
			b.telegram.SendMessage(
				message.Chat,
				"Sorry, I don't understand...",
				nil,
			)
			return nil
		}

		b.commandsCounter.WithLabelValues(text).Inc()
		handler(message)

		return nil
	}

	messages := make(chan telebot.Message, 100)
	b.telegram.Listen(messages, time.Second)

	var gr run.Group
	{
		gr.Add(func() error {
			return b.sendWebhook(ctx)
		}, func(err error) {
		})
	}
	{
		gr.Add(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case message := <-messages:
					if err := process(message); err != nil {
						level.Info(b.logger).Log(
							"msg", "failed to process message",
							"err", err,
							"sender_id", message.Sender.ID,
							"sender_username", message.Sender.Username,
						)
					}
				}
			}
		}, func(err error) {
		})
	}

	return gr.Run()
}

// sendWebhook sends messages received via webhook to all subscribed chats
func (b *Bot) sendWebhook(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
			//case w := <-webhooks:
			//	chats, err := b.chats.List()
			//	if err != nil {
			//		level.Error(b.logger).Log("msg", "failed to get chat list from store", "err", err)
			//		continue
			//	}
			//
			//	data := &template.Data{
			//		Receiver:          w.Receiver,
			//		Status:            w.Status,
			//		Alerts:            w.Alerts,
			//		GroupLabels:       w.GroupLabels,
			//		CommonLabels:      w.CommonLabels,
			//		CommonAnnotations: w.CommonAnnotations,
			//		ExternalURL:       w.ExternalURL,
			//	}
			//
			//	out, err := b.templates.ExecuteHTMLString(`{{ template "telegram.default" . }}`, data)
			//	if err != nil {
			//		level.Warn(b.logger).Log("msg", "failed to template alerts", "err", err)
			//		continue
			//	}
			//
			//	for _, chat := range chats {
			//		err = b.telegram.SendMessage(chat, b.truncateMessage(out), &telebot.SendOptions{ParseMode: telebot.ModeHTML})
			//		if err != nil {
			//			level.Warn(b.logger).Log("msg", "failed to send message to subscribed chat", "err", err)
			//		}
			//	}
		}
	}
}

func (b *Bot) handleStart(message telebot.Message) {
	if err := b.chats.Add(message.Chat); err != nil {
		level.Warn(b.logger).Log("msg", "failed to add chat to chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't add this chat to the subscribers list.", nil)
		return
	}

	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
	level.Info(b.logger).Log(
		"msg", "user subscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
}

func (b *Bot) handleStop(message telebot.Message) {
	if err := b.chats.Remove(message.Chat); err != nil {
		level.Warn(b.logger).Log("msg", "failed to remove chat from chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't remove this chat from the subscribers list.", nil)
		return
	}

	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
	level.Info(b.logger).Log(
		"msg", "user unsubscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
}

func (b *Bot) handleHelp(message telebot.Message) {
	b.telegram.SendMessage(message.Chat, responseHelp, nil)
}

func (b *Bot) handleChats(message telebot.Message) {
	chats, err := b.chats.List()
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to list chats from chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't list the subscribed chats.", nil)
		return
	}

	list := ""
	for _, chat := range chats {
		if chat.IsGroupChat() {
			list = list + fmt.Sprintf("@%s\n", chat.Title)
		} else {
			list = list + fmt.Sprintf("@%s\n", chat.Username)
		}
	}

	b.telegram.SendMessage(message.Chat, "Currently these chat have subscribed:\n"+list, nil)
}

func (b *Bot) handleStatus(message telebot.Message) {
	ok, err := b.alertmanager.General.GetStatus(nil)
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to get status", "err", err)
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to get status... %v", err), nil)
		return
	}

	value, err := ok.Payload.Uptime.Value()

	uptime := durafmt.Parse(time.Since(value.(time.Time)))
	uptimeBot := durafmt.Parse(time.Since(b.startTime))

	b.telegram.SendMessage(
		message.Chat,
		fmt.Sprintf(
			"*AlertManager*\nVersion: %s\nUptime: %s\n*AlertManager Bot*\nVersion: %s\nUptime: %s",
			*ok.Payload.VersionInfo.Version,
			uptime,
			b.revision,
			uptimeBot,
		),
		&telebot.SendOptions{ParseMode: telebot.ModeMarkdown},
	)
}

func (b *Bot) handleAlerts(message telebot.Message) {
	active := true
	getAlerts, err := b.alertmanager.Alert.GetAlerts(alert.NewGetAlertsParams().WithActive(&active))
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to list alerts... %v", err), nil)
		return
	}

	alerts := internalAlerts(getAlerts.Payload)

	if len(alerts) == 0 {
		b.telegram.SendMessage(message.Chat, "No alerts right now! 🎉", nil)
		return
	}

	out, err := b.tmplAlerts(alerts...)
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to template alerts: %w", err), nil)
		return
	}

	err = b.telegram.SendMessage(message.Chat, b.truncateMessage(out), &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to send message", "err", err)
	}
}

func internalAlerts(getAlerts models.GettableAlerts) []*types.Alert {
	alerts := make([]*types.Alert, 0, len(getAlerts))
	for _, a := range getAlerts {
		alert := &types.Alert{
			Alert: model.Alert{
				Labels: model.LabelSet{},
				//Annotations:  nil, TODO!!!
				StartsAt:     time.Time(*a.StartsAt),
				EndsAt:       time.Time(*a.EndsAt),
				GeneratorURL: string(a.GeneratorURL),
			},
			UpdatedAt: time.Time(*a.UpdatedAt),
			//Timeout:   false, TODO!!!
		}
		for key, value := range a.Labels {
			alert.Labels[model.LabelName(key)] = model.LabelValue(value)
		}
		alerts = append(alerts, alert)
	}
	return alerts
}

func (b *Bot) handleSilences(message telebot.Message) {
	getSilences, err := b.alertmanager.Silence.GetSilences(silence.NewGetSilencesParams())
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to list silences... %v", err), nil)
		return
	}

	if len(getSilences.Payload) == 0 {
		b.telegram.SendMessage(message.Chat, "No silences right now.", nil)
		return
	}

	var out string
	for _, silence := range getSilences.Payload {
		s := types.Silence{
			ID:        *silence.ID,
			StartsAt:  time.Time(*silence.StartsAt),
			EndsAt:    time.Time(*silence.EndsAt),
			UpdatedAt: time.Time(*silence.UpdatedAt),
			CreatedBy: *silence.CreatedBy,
			Comment:   *silence.Comment,
		}
		for _, m := range s.Matchers {
			s.Matchers = append(s.Matchers, &types.Matcher{
				Name:    m.Name,
				Value:   m.Value,
				IsRegex: m.IsRegex,
			})
		}
		switch *silence.Status.State {
		case string(types.SilenceStatePending):
			s.Status.State = types.SilenceStatePending
		case string(types.SilenceStateActive):
			s.Status.State = types.SilenceStateActive
		case string(types.SilenceStateExpired):
			s.Status.State = types.SilenceStateExpired
		}

		out = out + alertmanager.SilenceMessage(s) + "\n"
	}

	b.telegram.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (b *Bot) tmplAlerts(alerts ...*types.Alert) (string, error) {
	data := b.templates.Data("default", nil, alerts...)

	out, err := b.templates.ExecuteHTMLString(`{{ template "telegram.default" . }}`, data)
	if err != nil {
		return "", err
	}

	return out, nil
}

// Truncate very big message
func (b *Bot) truncateMessage(str string) string {
	truncateMsg := str
	if len(str) > 4095 { // telegram API can only support 4096 bytes per message
		level.Warn(b.logger).Log("msg", "Message is bigger than 4095, truncate...")
		// find the end of last alert, we do not want break the html tags
		i := strings.LastIndex(str[0:4080], "\n\n") // 4080 + "\n<b>[SNIP]</b>" == 4095
		if i > 1 {
			truncateMsg = str[0:i] + "\n<b>[SNIP]</b>"
		} else {
			truncateMsg = "Message is too long... can't send.."
			level.Warn(b.logger).Log("msg", "truncateMessage: Unable to find the end of last alert.")
		}
		return truncateMsg
	}
	return truncateMsg
}
