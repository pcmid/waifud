package downloader

import (
	"context"
	"github.com/pcmid/waifud/core"
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
	rpcUrl    string
	rpcSecret string

	missions map[string]*Mission
	sms      chan core.Message
	rms      chan *Mission
}

type Mission struct {
	Gid string
}

func (a *Aria2c) Name() string {
	return "aria2c"
}

func (a *Aria2c) ListeningTypes() []string {
	return []string{
		"enclosure",
	}
}

func (a *Aria2c) Init() {

	a.rpcUrl = "http://127.0.0.1:6800/jsonrpc"
	a.rpcSecret = ""

	a.missions = make(map[string]*Mission)

	a.rms = make(chan *Mission)

	if viper.IsSet("service.aria2c.url") {
		a.rpcUrl = viper.GetString("service.aria2c.url")
		log.Tracef("set aria2c rpc url %s", a.rpcUrl)
	} else {
		log.Warnf("aria2 rpc url not found, use %s", a.rpcUrl)
	}

	if viper.IsSet("service.aria2c.secret") {
		a.rpcSecret = viper.GetString("service.aria2c.secret")
		log.Tracef("set aria2c rpc secret %s", a.rpcSecret)

	} else {
		log.Warnf("aria2 rpc secret not found, use \"%s\"", a.rpcSecret)
	}
}

func (a *Aria2c) SetMessageChan(ms chan core.Message) {
	a.sms = ms
}

func (a *Aria2c) Send(message core.Message) {
	a.sms <- message
}

func (a *Aria2c) Serve() {
	rpcc, _ := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)
	tick := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-tick.C:
			for gid, url := range a.missions {
				status, _ := rpcc.TellStatus(gid)

				if status.Status == "complete" {
					log.Infof("%s download complete %s", a.Name(), url)

					if followed := status.FollowedBy; followed != nil {

						for _, g := range followed {
							a.missions[g] = &Mission{
								Gid: g,
							}
						}
					} else {
						for _, file := range status.Files {
							a.Send(core.Message{
								Type: "notify",
								Msg:  file.Path[strings.LastIndex(file.Path, "/")+1:],
							})
						}
					}
					delete(a.missions, gid)
				}

			}

		case mission := <-a.rms:
			a.missions[mission.Gid] = mission
		}
	}

}

func (a *Aria2c) Handle(message core.Message) {
	url := message.Message().(string)
	a.Download(url)
}

func (a *Aria2c) Download(url string) {
	log.Infof("%s Download %s", a.Name(), url)

	rpcc, err := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)
	defer func() {
		_ = rpcc.Close()
	}()

	if err != nil {
		log.Errorf("%s Failed to connect aria2 rpc server: %s", a.Name(), err)
		return
	}

	gid, err := rpcc.AddURI(url, rpc.Option{})

	if err != nil {
		log.Errorf("%s Failed to AddURL for aria2c: %s", a.Name(), err)
		return
	}

	//log.Trace(Gid)

	a.rms <- &Mission{
		Gid: gid,
	}
}
