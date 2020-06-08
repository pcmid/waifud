package services

import (
	"fmt"
	"github.com/pcmid/waifud/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"strings"
	"time"
)

func init() {
	core.Register(&TeleBot{})
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
		"status",
	}
}

func (t *TeleBot) Init() {
	token := viper.GetString("service.teleBot.token")

	if token == "" {
		log.Error("TeleBot token not found, exit")
		os.Exit(-1)
	}

	log.Tracef("set telebot token %s", token)

	b, err := tb.NewBot(tb.Settings{
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
	b.Handle("/link", t.commandLink)
	b.Handle("/status", t.commandStatus)

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

	switch message.Type() {
	case "notify":
		go t.notify(message.Get("content").(string), false)
	case "feeds":
		feeds := message.Get("feeds").([]*Feed)
		if len(feeds) == 0 {
			go t.notify("未找到订阅", false)
			return
		}

		resp := strings.Builder{}
		resp.WriteString("订阅如下:\n")

		for _, feed := range feeds {
			resp.WriteString(fmt.Sprintf("[%s](%s)\n", feed.Title, feed.URL))
		}

		go t.notify(resp.String(), true)

	case "status":
		statues := message.Get("missions").(map[string]*Mission)
		if len(statues) == 0 {
			go t.notify("未找到下载项目", false)
			return
		}

		resp := strings.Builder{}
		resp.WriteString("正在下载:\n")
		items := 0

		for _, status := range statues {
			resp.WriteString(
				fmt.Sprintf("名称: %s\n\t进度: %.2f%%\n", status.Name, status.ProgressRate*100),
			)
			items++

			if items >= 50 {
				go t.notify(resp.String(), false)
				resp.Reset()
				items = 0
			}
		}

		if resp.Len() != 0 {
			go t.notify(resp.String(), false)

		}
	}
}

func (t *TeleBot) notify(m string, isMarkDown bool) {
	retryTimes := 10
	tc := time.Tick(30 * time.Second)

	opt := &tb.SendOptions{
		ReplyTo:               nil,
		ReplyMarkup:           nil,
		DisableWebPagePreview: true,
		DisableNotification:   false,
		ParseMode:             "",
	}

	if isMarkDown {
		opt.ParseMode = tb.ModeMarkdown
	}

	for {
		if retryTimes == 0 {
			return
		}
		if _, e := t.bot.Send(t.chat, m, opt); e == nil {
			return
		} else {
			log.Errorf("Failed to send message: %s, retrying...", e)
			retryTimes--
		}
		<-tc
	}
}

func (t *TeleBot) commandSub(m *tb.Message) {
	payload := m.Payload
	if payload == "" {
		_, _ = t.bot.Send(m.Sender, "usage :/sub URL [dir]")
		return
	}
	log.Trace(payload)

	contents := strings.Split(payload, " ")

	var url string
	var dir string

	url = contents[0]
	if len(contents) > 1 {
		dir = contents[1]
	}

	t.chat = m.Sender

	t.Send(
		core.NewMessage("subscription").
			Set("content", url).
			Set("operation", Sub).
			Set("dir", dir),
	)
}

func (t *TeleBot) commandUnSub(m *tb.Message) {
	url := m.Payload
	if url == "" {
		_, _ = t.bot.Send(m.Sender, "usage :/unsub URL")
		return
	}
	log.Trace(url)

	t.chat = m.Sender

	msg := core.NewMessage("subscription").
		Set("content", url).
		Set("operation", UnSub)

	t.Send(msg)
}

func (t *TeleBot) commandGetSub(m *tb.Message) {
	t.chat = m.Sender

	msg := core.NewMessage("subscription").
		Set("operation", GetSub)

	t.Send(msg)
}

func (t *TeleBot) commandLink(m *tb.Message) {
	t.chat = m.Sender

	payload := m.Payload
	log.Trace(payload)

	contents := strings.Split(payload, " ")

	var url string
	var dir string

	url = contents[0]
	if len(contents) > 1 {
		dir = contents[1]
	}

	msg := core.NewMessage("link").
		Set("url", url).
		Set("dir", dir)

	t.Send(msg)
}

func (t *TeleBot) commandStatus(m *tb.Message) {
	t.chat = m.Sender

	t.Send(core.NewMessage("aria2c_api").
		Set("content", "status"),
	)
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
		} else {
			log.Info("Failed to init telebot, wait...")
		}
	}
}

func (t *TeleBot) SetMessageChan(ms chan core.Message) {
	t.sms = ms
}

func (t *TeleBot) Send(message core.Message) {
	t.sms <- message
}
