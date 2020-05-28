package services

import (
	"context"
	"fmt"
	"github.com/pcmid/waifud/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/zyxar/argo/rpc"
	"strconv"
	"strings"
	"time"
)

func init() {
	core.Register(&Aria2c{})
}

type Aria2c struct {
	rpcUrl    string
	rpcSecret string

	missions    map[string]*Mission
	newMissions chan *Mission

	sms chan core.Message

	globalDir string
}

type Mission struct {
	Gid          string
	Name         string
	Status       string
	ProgressRate float64
	FollowedBy   []string
}

func (a *Aria2c) Name() string {
	return "aria2c"
}

func (a *Aria2c) ListeningTypes() []string {
	return []string{
		"item",
		"aria2c_api",
	}
}

func (a *Aria2c) Init() {

	a.rpcUrl = "http://127.0.0.1:6800/jsonrpc"
	a.rpcSecret = ""

	a.missions = make(map[string]*Mission)

	a.newMissions = make(chan *Mission)

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

	a.getGlobalDir()
}

func (a *Aria2c) SetMessageChan(ms chan core.Message) {
	a.sms = ms
}

func (a *Aria2c) Send(message core.Message) {
	a.sms <- message
}

func (a *Aria2c) Serve() {
	tick := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-tick.C:
			a.UpdateStatus()

			for gid, m := range a.missions {
				switch m.Status {
				case "complete":
					log.Infof("%s download completed", m.Name)
					a.Send(
						core.NewMessage("notify").
							Set("content", fmt.Sprintf("%s 下载完成", m.Name)),
					)
					if followed := m.FollowedBy; followed != nil {
						for _, g := range followed {
							go a.addMission(g)
						}
					}
					delete(a.missions, gid)

				case "error":
					a.Send(
						core.NewMessage("notify").
							Set("content", fmt.Sprintf("%s 下载失败", m.Name)),
					)
					delete(a.missions, gid)

				case "removed":
					delete(a.missions, gid)
				}
			}
		case mission := <-a.newMissions:
			a.missions[mission.Gid] = mission
		}
	}
}

func (a *Aria2c) Handle(message core.Message) {

	switch message.Type() {
	case "aria2c_api":
		method := message.Get("content")
		switch method {
		case "status":

			m := core.NewMessage("status").
				Set("missions", a.missions)

			a.Send(m)
		}
	case "item":
		url := message.Get("content").(string)
		dir := message.Get("dir").(string)
		a.Download(url, dir)
	}
}

func (a *Aria2c) Download(url, dir string) {
	log.Infof("%s Download %s", a.Name(), url)

	rpcc, err := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)
	defer func() {
		_ = rpcc.Close()
	}()

	if err != nil {
		log.Errorf("%s Failed to connect aria2 rpc server: %s", a.Name(), err)
		a.Send(
			core.NewMessage("notify").
				Set("content", "添加下载失败"),
		)
		return
	}

	gid, err := rpcc.AddURI(url, rpc.Option{
		"dir": a.globalDir + "/" + dir,
	})

	if err != nil {
		log.Errorf("%s Failed to AddURL for aria2c: %s", a.Name(), err)
		a.Send(
			core.NewMessage("notify").
				Set("content", "添加下载失败"),
		)
		return
	}

	a.addMission(gid)
}

func (a *Aria2c) UpdateStatus() {
	rpcc, _ := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)

	for gid, mission := range a.missions {
		s, _ := rpcc.TellStatus(gid)
		mission.Status = s.Status
		mission.FollowedBy = s.FollowedBy
		if s.InfoHash == "" {
			// file from url
			mission.Name = s.Files[0].Path[strings.LastIndex(s.Files[0].Path, "/")+1:]
		} else if s.BitTorrent.Info.Name != "" {
			// files from torrent
			mission.Name = s.BitTorrent.Info.Name
		} else {
			// file from metalink
			mission.Name = s.Files[0].Path
		}

		completedLength, _ := strconv.ParseFloat(s.CompletedLength, 10)
		totalLength, _ := strconv.ParseFloat(s.TotalLength, 10)

		if s.TotalLength == "0" {
			mission.ProgressRate = 0
			return
		}
		mission.ProgressRate = completedLength / totalLength
	}
}

func (a *Aria2c) getGlobalDir() {
	rpcc, _ := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)

	m, err := rpcc.GetGlobalOption()

	if err != nil {
		log.Error(err)
		return
	}

	log.Trace(m)

	a.globalDir = m["dir"].(string)
}

func (a *Aria2c) addMission(gid string) {
	a.newMissions <- &Mission{
		Gid: gid,
	}
}