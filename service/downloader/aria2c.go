package downloader

import (
	"context"
	"github.com/pcmid/waifud/messages"
	log "github.com/sirupsen/logrus"
	"github.com/zyxar/argo/rpc"
	"time"
)

type Aria2c struct {
	BaseDownloader

	rpcUrl    string
	rpcSecret string
}

func (a *Aria2c) Init() {
	a.rpcUrl = "http://127.0.0.1:6800/jsonrpc"
	a.rpcSecret = "s"

}

func (a *Aria2c) Name() string {
	return "aria2c"
}

func (a *Aria2c) SetMessageChan(chan messages.Message) {
	//panic("implement me")
}

func (a *Aria2c) Send(message messages.Message) {
	panic("implement me")
}

func (a *Aria2c) Serve() {
	//panic("implement me")
}

func (a *Aria2c) Handle(message messages.Message) {
	_url := message.(*messages.DLMessage).URL
	a.Download(_url)
}

func (a *Aria2c) Download(_url string) {
	log.Infof("Download %s", _url)

	rpcc, err := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)
	defer func() {
		rpcc.Close()
	}()

	if err != nil {
		log.Errorf("Failed to connect aria2 rpc server :%s", err)
		return
	}

	gid, err := rpcc.AddURI(_url, rpc.Option{})

	if err != nil {
		log.Errorf("Failed to AddURL for aria2c :%s", err)
		return
	}

	log.Trace(gid)

}
