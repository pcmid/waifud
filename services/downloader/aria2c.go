package downloader

import (
	"context"
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/services"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/zyxar/argo/rpc"
	"strings"
	"time"
)

func init() {
	services.ServiceMap["aria2c"] = &Aria2c{}
}

type Aria2c struct {
	BaseDownloader

	rpcUrl    string
	rpcSecret string

	tasks map[string]string
	sms   chan messages.Message
	rms   chan struct {
		G string
		U string
	}
}

func (a *Aria2c) Types() []string {
	//panic("implement me")
	return []string{a.Type()}
}

func (a *Aria2c) Init() {

	a.rpcUrl = "http://127.0.0.1:6800/jsonrpc"
	a.rpcSecret = ""

	a.tasks = make(map[string]string)

	a.rms = make(chan struct {
		G string
		U string
	})

	if viper.IsSet("services.aria2c.url") {
		a.rpcUrl = viper.GetString("services.aria2c.url")
	} else {
		log.Warnf("aria2 rpc url not found, use %s", a.rpcUrl)
	}

	if viper.IsSet("services.aria2c.secret") {
		a.rpcSecret = viper.GetString("services.aria2c.secret")
	} else {
		log.Warnf("aria2 rpc secret not found, use \"%s\"", a.rpcSecret)
	}
}

func (a *Aria2c) Name() string {
	return "aria2c"
}

func (a *Aria2c) SetMessageChan(ms chan messages.Message) {
	//panic("implement me")
	a.sms = ms
}

func (a *Aria2c) Send(message messages.Message) {
	//panic("implement me")
	a.sms <- message
}

func (a *Aria2c) Serve() {
	//panic("implement me")

	rpcc, _ := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)

	tick := time.NewTicker(2 * time.Second)

	for {

		select {
		case <-tick.C:
			for gid, _url := range a.tasks {
				status, _ := rpcc.TellStatus(gid)

				if status.Status == "complete" {
					log.Infof("%s download complete %s", a.Name(), _url)

					if followed := status.FollowedBy; followed != nil {

						for _, g := range followed {
							a.tasks[g] = ""
						}
					} else {
						for _, file := range status.Files {
							a.Send(&messages.ResultMessage{
								Status: "complete",
								Msg:    file.Path[strings.LastIndex(file.Path, "/")+1:],
							})
						}
					}

					delete(a.tasks, gid)
				}

			}

		case newTask := <-a.rms:
			a.tasks[newTask.G] = newTask.U
		}
	}

}

func (a *Aria2c) Handle(message messages.Message) {
	_url := message.(*messages.DLMessage).URL
	a.Download(_url)
}

func (a *Aria2c) Download(_url string) {
	log.Infof("%s Download %s", a.Name(), _url)

	rpcc, err := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)
	defer func() {
		rpcc.Close()
	}()

	if err != nil {
		log.Errorf("%s Failed to connect aria2 rpc server: %s", a.Name(), err)
		return
	}

	gid, err := rpcc.AddURI(_url, rpc.Option{})

	if err != nil {
		log.Errorf("%s Failed to AddURL for aria2c: %s", a.Name(), err)
		return
	}

	//log.Trace(Gid)

	a.rms <- struct {
		G string
		U string
	}{G: gid, U: _url}

}
