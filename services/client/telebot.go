package client

import (
	"fmt"
	"github.com/pcmid/waifud/core"
	"github.com/pcmid/waifud/services"
	"github.com/pcmid/waifud/services/database"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"strings"
	"time"
)

func init() {
	services.ServiceMap["telebot"] = &TeleBot{}
}

type TeleBot struct {
	bot *tb.Bot

	rms chan core.Message
	sms chan core.Message

	chat tb.Recipient
}

func (t *TeleBot) Name() string {
	return "telebot"
}

func (t *TeleBot) ListeningTypes() []string {
	return []string{
		"feeds",
		"notify",
	}
}

func (t *TeleBot) Init() {
	token := viper.GetString("service.TeleBot.token")

	if token == "" {
		log.Error("TeleBot token not found, exit")
		os.Exit(-1)
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

	b.Handle("/sub", t.commandSub)
	b.Handle("/unsub", t.commandUnSub)
	b.Handle("/getsub", t.commandGetSub)

	b.Handle("/a", func(m *tb.Message) {
		fmt.Printf("%#v", m)
	})

	t.bot = b
}

func (t *TeleBot) Serve() {
	if t.bot == nil {
		log.Errorf("Failed to start %s", t.Name())
		return
	}
	t.bot.Start()
}

func (t *TeleBot) Handle(message core.Message) {
	if t.chat == nil {
		return
	}

	switch message.Type {
	case "notify":
		_, _ = t.bot.Send(t.chat, message.Message().(string))
	case "feeds":
		feeds := message.Message().(map[string]*database.Feed)
		if len(feeds) == 0 {
			_, _ = t.bot.Send(t.chat, "未找到订阅!")
			return
		}

		resp := strings.Builder{}
		resp.WriteString("订阅如下:\n")

		for _, feed := range feeds {
			resp.WriteString(fmt.Sprintf("[%s](%s)\n", feed.Title, feed.URL))
		}
		_, _ = t.bot.Send(t.chat, resp.String(), &tb.SendOptions{
			ReplyTo:               nil,
			ReplyMarkup:           nil,
			DisableWebPagePreview: true,
			DisableNotification:   false,
			ParseMode:             tb.ModeMarkdown,
		})
	}
}

func (t *TeleBot) commandSub(m *tb.Message) {
	url := m.Payload
	if url == "" {
		_, _ = t.bot.Send(m.Sender, "useage :/sub URL")
		return
	}
	log.Trace(url)

	t.chat = m.Chat

	t.Send(core.Message{
		Type: "subscription",
		Msg: &database.Subscription{
			Op:  database.Sub,
			Url: url,
		},
	})
}

func (t *TeleBot) commandUnSub(m *tb.Message) {
	url := m.Payload
	if url == "" {
		_, _ = t.bot.Send(m.Sender, "useage :/unsub URL")
		return
	}
	log.Trace(url)

	t.chat = m.Chat

	t.Send(core.Message{
		Type: "subscription",
		Msg: &database.Subscription{
			Op:  database.UnSub,
			Url: url,
		},
	})
}

func (t *TeleBot) commandGetSub(m *tb.Message) {
	t.chat = m.Chat

	t.Send(core.Message{
		Type: "subscription",
		Msg: &database.Subscription{
			Op:  database.GetSub,
			Url: "",
		},
	})
}

func (t *TeleBot) initAfterFailed(token string) *tb.Bot {
	tc := time.Tick(30 * time.Second)
	for {
		<-tc
		b, err := tb.NewBot(tb.Settings{
			Token:  token,
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})

		if err == nil {
			log.Info("Init telebot successfully")
			return b
		}
	}
}

func (t *TeleBot) SetMessageChan(ms chan core.Message) {
	t.sms = ms
}

func (t *TeleBot) Send(message core.Message) {
	t.sms <- message
}
