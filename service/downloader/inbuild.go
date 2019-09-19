package downloader

import (
	"github.com/pcmid/waifud/messages"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)
import log "github.com/sirupsen/logrus"

// the inbiuld.go just for test, dont use it

type InBuilt struct {
	BaseDownloader
}

func (i *InBuilt) Init() {
	//panic("implement me")
}

func (i *InBuilt) Name() string {
	return "InBuilt"
}

func (i *InBuilt) SetMessageChan(chan messages.Message) {
	//panic("implement me")
}

func (i *InBuilt) Send(message messages.Message) {
	panic("implement me")
}

func (i *InBuilt) Serve() {
	//panic("implement me")
}

func (i *InBuilt) Handle(message messages.Message) {
	_url := message.(*messages.DLMessage).URL
	i.Download(_url)
}

func (i *InBuilt) Download(_url string) {
	log.Infof("%s Download %s", i.Name(), _url)
	resp, _ := http.Get(_url)

	body, _ := ioutil.ReadAll(resp.Body)
	u, _ := url.Parse(_url)

	filename := "/tmp/" + u.Path[strings.LastIndex(u.Path, "/")+1:]

	_ = ioutil.WriteFile(filename, body, 0644)
}
