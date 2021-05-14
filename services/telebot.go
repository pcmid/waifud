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
	bot  *tb.Bot
	chat tb.Recipient
	//isPrivate bool
	core.Receiver
	core.Sender
}

func (t *TeleBot) Name() string {
	return "telebot"
}

func (t *TeleBot) ListeningTypes() []string {
	return []string{
		"notify",
	}
}

func (t *TeleBot) Init() {
	token := viper.GetString("service.telebot.token")
	if token == "" {
		log.Error("TeleBot token not found, exit")
		os.Exit(-1)
	}
	log.Tracef("set telebot token %s", token)
	//t.isPrivate = viper.GetBool("service.telebot.private")
	//if t.isPrivate == false {
	//	log.Warnf("telebot.private is been set false, everyone can access your bot!")
	//}
	//log.Tracef("set telebot private: %v", t.isPrivate)
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Errorf("Failed to init telebot: %s", err)
		b = t.initAfterFailed(token)
	}
	{
		b.Handle("/ping", func(m *tb.Message) {
			_, _ = b.Send(m.Sender, "pong!")
		})
		b.Handle("/reg", t.commandReg)
		b.Handle("/sub", t.commandSub)
		b.Handle("/unsub", t.commandUnSub)
		b.Handle("/getsub", t.commandGetSub)
		b.Handle("/link", t.commandLink)
		b.Handle("/status", t.commandStatus)
	}
	t.bot = b
}

func (t *TeleBot) Serve() {
	if t.bot == nil {
		log.Errorf("Failed to start %s", t.Name())
		return
	}
	t.bot.Start()
}

func (t *TeleBot) Start() {
	t.Init()
	t.Serve()
}

func (t *TeleBot) Handle(message core.Message) {
	if t.chat == nil {
		return
	}
	t.Receiver.Handle(message)
	switch message.Type() {
	case "notify":
		t.notify(message.Get("content").(string), false)
	}
}

func (t *TeleBot) notify(m string, isMarkDown bool) {
	retryTimes := 3
	tc := time.Tick(60 * time.Second)
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
	go func() {
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
	}()
}

func (t *TeleBot) commandReg(m *tb.Message) {
	if t.chat == nil {
		log.Infof("new user registered: %s", m.Sender.Username)
		t.chat = m.Sender
		t.notify("注册成功", false)
	} else {
		return
	}
}

func (t *TeleBot) commandSub(m *tb.Message) {
	if !t.check(m) {
		return
	}
	payload := m.Payload
	if payload == "" {
		_, _ = t.bot.Send(m.Sender, "usage :/sub URL [tag]")
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
	resp := t.Send(core.NewMessage("subscription").
		Set("content", url).
		Set("operation", Sub).
		Set("tag", dir),
	).WaitResponse()
	t.notify(resp.Get("message").(string), false)
}

func (t *TeleBot) commandUnSub(m *tb.Message) {
	if !t.check(m) {
		return
	}
	url := m.Payload
	if url == "" {
		_, _ = t.bot.Send(m.Sender, "usage :/unsub URL")
		return
	}
	log.Trace(url)
	resp := t.Send(core.NewMessage("subscription").
		Set("content", url).
		Set("operation", UnSub),
	).WaitResponse()
	t.notify(resp.Get("message").(string), false)
}

func (t *TeleBot) commandGetSub(m *tb.Message) {
	if !t.check(m) {
		return
	}
	response := t.Send(core.NewMessage("subscription").
		Set("operation", GetSub),
	).WaitResponse()
	feeds := response.Get("feeds").([]*Feed)
	if len(feeds) == 0 {
		t.notify("未找到订阅", false)
		return
	}
	resp := strings.Builder{}
	resp.WriteString("订阅如下:\n")
	for _, feed := range feeds {
		resp.WriteString(fmt.Sprintf("[%s](%s)\n", feed.Title, feed.Tag))
	}
	t.notify(resp.String(), true)
}

func (t *TeleBot) commandLink(m *tb.Message) {
	if !t.check(m) {
		return
	}
	payload := m.Payload
	log.Trace(payload)
	contents := strings.Split(payload, " ")
	var url string
	var dir string
	url = contents[0]
	if len(contents) > 1 {
		dir = contents[1]
	}
	resp := t.Send(core.NewMessage("link").
		Set("url", url).
		Set("dir", dir),
	).WaitResponse()
	t.notify(resp.Get("message").(string), false)
}

func (t *TeleBot) commandStatus(m *tb.Message) {
	if !t.check(m) {
		return
	}
	statues := t.Send(core.NewMessage("aria2c_api").
		Set("content", "status"),
	).WaitResponse().Get("missions")

	if !(statues != nil && len(statues.(map[string]*Mission)) != 0) {
		t.notify("未找到下载项目", false)
		return
	}
	resp := strings.Builder{}
	resp.WriteString("正在下载:\n")
	items := 0
	for _, status := range statues.(map[string]*Mission) {
		resp.WriteString(
			fmt.Sprintf("名称: %s\n\t进度: %.2f%%\n", status.Name, status.ProgressRate*100),
		)
		items++
		if items >= 50 {
			t.notify(resp.String(), false)
			resp.Reset()
			items = 0
		}
	}
	if resp.Len() != 0 {
		t.notify(resp.String(), false)
	}
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

func (t *TeleBot) check(m *tb.Message) bool {
	//if !t.isPrivate {
	//	t.chat = m.Sender
	//	return true
	//}
	//if t.chat != nil {
	//	log.Tracef("chat: %s", t.chat.Recipient())
	//}
	//log.Tracef("sender: %s", m.Sender.Recipient())
	//if t.isPrivate && t.chat != nil && t.chat.Recipient() == m.Sender.Recipient() {
	//	return true
	//}
	//log.Warnf("unauthorized access: %v", m.Sender)
	//_, _ = t.bot.Send(m.Sender, "未授权的访问!")
	//return false
	t.chat = m.Sender
	return true
}
