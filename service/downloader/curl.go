package downloader

import (
	"github.com/pcmid/waifud/messages"
	log "github.com/sirupsen/logrus"
)

type Curl struct {
	BaseDownloader
}

func (c *Curl) Init() {
	//panic("implement me")
}

func (c *Curl) Name() string {
	return "curl"
}

func (c *Curl) SetMessageChan(chan messages.Message) {
	//panic("implement me")
}

func (c *Curl) Send(message messages.Message) {
	panic("implement me")
}

func (c *Curl) Serve() {
	//panic("implement me")
}

func (c *Curl) Describe() string {
	return "curl"
}

func (c *Curl) Handle(message messages.Message) {
	url := message.(*messages.DLMessage).URL
	c.Download(url)
}

func (c *Curl) Download(url string) {
	log.Infof("Download %s", url)
}
