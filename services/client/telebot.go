package client

import (
	"fmt"
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/services"
	"github.com/pcmid/waifud/services/database"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"
	"time"
)

func init() {
	services.ServiceMap["telebot"] = &TeleBot{}
}

type TeleBot struct {
	BaseClient
	bot *tb.Bot
	sms chan messages.Message

	chat tb.Recipient
}

func (t *TeleBot) commadSub(m *tb.Message) {
	feedUrl := m.Payload
	if feedUrl == "" {
		_, _ = t.bot.Send(m.Sender, "useage :/sub URL")
		return
	}
	log.Trace(feedUrl)

	t.chat = m.Chat

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

func (t *TeleBot) Types() []string {
	//panic("implement me")
	return []string{"client", "notifier"}
}

func (t *TeleBot) initAfterFailed(token string) *tb.Bot {
	tc := time.Tick(30 * time.Second)
	for  {
		<- tc
		b, err := tb.NewBot(tb.Settings{
			// the token just for test
			Token:  token,
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})

		if err == nil {
			log.Info("Init telebot successfully")
			return b
		}
	}
}

func (t * TeleBot)handleFunc(	)  {

}

func (t *TeleBot) Init() {
	//panic("implement me")

	token := viper.GetString("service.TeleBot.token")

	if token == "" {
		log.Error("TeleBot token not found")
		return
	}

	log.Tracef("set telebot token %s", token)

	b, err := tb.NewBot(tb.Settings{
		// the token just for test
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Errorf("Failed to init telebot: %s", err)
		b = t.initAfterFailed(token)
	}

	b.Handle("/ping", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "pong!")
	})

	b.Handle("/sub", t.commadSub)
	b.Handle("/unsub", t.commadUnsub)

	b.Handle("/a", func(m *tb.Message) {
		fmt.Printf("%#v", m)
	})

	t.bot = b
}

func (t *TeleBot) Serve() {
	//panic("implement me")
	if t.bot == nil {
		log.Errorf("Failed to start %s", t.Name())
		return
	}
	t.bot.Start()
}

func (t *TeleBot) Handle(message messages.Message) {
	//panic("implement me")

	if t.chat == nil {
		return
	}

	msg := message.(*messages.ResultMessage)

	_, _ = t.bot.Send(t.chat, "download complete: "+msg.Msg)
}

func (t *TeleBot) SetMessageChan(ms chan messages.Message) {
	//panic("implement me")
	t.sms = ms
}

func (t *TeleBot) Send(message messages.Message) {
	//panic("implement me")
	t.sms <- message
}
