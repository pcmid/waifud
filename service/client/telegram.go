package client

import (
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/service/database"
	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v2"
	"time"
)

type TeleBot struct {
	BaseClient
	bot *tb.Bot
	sms chan messages.Message
}

func (t *TeleBot) commadSub(m *tb.Message) {
	feedUrl := m.Payload
	if feedUrl == "" {
		_, _ = t.bot.Send(m.Sender, "useage :/sub URL")
		return
	}
	log.Trace(feedUrl)
	t.Send(&messages.DBMessage{
		Code: database.AddFeed,
		URL:  feedUrl,
	})
}

func (t *TeleBot) commadUnsub(m *tb.Message) {
	feedUrl := m.Payload
	if feedUrl == "" {
		_, _ = t.bot.Send(m.Sender, "useage :/unsub URL")
		return
	}
	log.Trace(feedUrl)
	t.Send(&messages.DBMessage{
		Code: database.DelFeed,
		URL:  feedUrl,
	})
}

func (t *TeleBot) Name() string {
	//panic("implement me")
	return "telebot"
}

func (t *TeleBot) Init() {
	//panic("implement me")
	b, err := tb.NewBot(tb.Settings{
		// the token just for test
		Token:  "754444894:AAFsW4v5gX875-CccWUsOxYftA_a5mG-gug",
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Errorf("Failed to init telebot :%s", err)
	}

	b.Handle("/hello", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "hello world")
	})

	b.Handle("/sub", t.commadSub)
	b.Handle("/unsub", t.commadUnsub)

	t.bot = b
}

func (t *TeleBot) Serve() {
	//panic("implement me")
	t.bot.Start()
}

func (t *TeleBot) Handle(message messages.Message) {
	panic("implement me")
}

func (t *TeleBot) SetMessageChan(ms chan messages.Message) {
	//panic("implement me")
	t.sms = ms
}

func (t *TeleBot) Send(message messages.Message) {
	//panic("implement me")
	t.sms <- message
}
