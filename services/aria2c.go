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
	"sync"
	"time"
)

func init() {
	core.Register(&Aria2c{})
}

type Aria2c struct {
	rpcUrl    string
	rpcSecret string

	missions map[string]*Mission

	sync.Mutex

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
	tick := time.NewTicker(10 * time.Second)

	for {
		<-tick.C
		a.update()
		a.check()
	}
}

func (a *Aria2c) Handle(message core.Message) {
	switch message.Type() {
	case "aria2c_api":
		method := message.Get("content")
		switch method {
		case "status":

			missions := make(map[string]*Mission)

			for _, mission := range a.missions {

				if mission.Status == "" {
					continue
				}

				missions[mission.Gid] = &Mission{
					Name:         mission.Name,
					ProgressRate: mission.ProgressRate,
				}
			}

			m := core.NewMessage("status").
				Set("missions", missions)

			a.Send(m)
		}
	case "item":
		url := message.Get("content").(string)
		dir := message.Get("dir").(string)
		a.Download(url, dir)

		// slow down for next
		time.Sleep(100 * time.Microsecond)
	}
}

func (a *Aria2c) Download(url, dir string) {
	log.Infof("download %s at %s", url, a.globalDir+"/"+dir)

	rpcc, err := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)
	defer func() {
		_ = rpcc.Close()
	}()

	if err != nil {
		log.Errorf("Failed to connect aria2 rpc server: %s", err)
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
		log.Errorf("Failed to AddURL for aria2c: %s", err)
		a.Send(
			core.NewMessage("notify").
				Set("content", "添加下载失败"),
		)
		return
	}

	a.addMission(gid)
}

func (a *Aria2c) update() {
	rpcc, _ := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)

	for gid, mission := range a.missions {
		s, err := rpcc.TellStatus(gid)

		if err != nil {
			log.Errorf("Failed to get status for %s: %s", mission.Name, err)
			mission.Status = "error"
			continue
		}

		if s.Status == "" {
			continue
		}

		mission.Status = s.Status

		mission.FollowedBy = s.FollowedBy

		if s.InfoHash == "" {
			// from url
			mission.Name = s.Files[0].Path[strings.LastIndex(s.Files[0].Path, "/")+1:]
		} else if s.BitTorrent.Info.Name != "" {
			// from torrent
			mission.Name = s.BitTorrent.Info.Name
		} else {
			// from torrent link
			mission.Name = "[METADATA]" + s.InfoHash
		}

		completedLength, _ := strconv.ParseFloat(s.CompletedLength, 10)
		totalLength, _ := strconv.ParseFloat(s.TotalLength, 10)

		if s.TotalLength == "0" {
			mission.ProgressRate = 0
			return
		} else {
			mission.ProgressRate = completedLength / totalLength
		}

		if completedLength == totalLength {
			mission.Status = "complete"
		}

		// slow down for next
		time.Sleep(100 * time.Microsecond)
	}
}

func (a *Aria2c) check() {
	for gid, m := range a.missions {
		switch m.Status {
		case "complete":
			log.Infof("%s download completed", m.Name)
			if m.FollowedBy != nil {
				for _, g := range m.FollowedBy {
					a.addMission(g)
				}
			} else {
				a.Send(
					core.NewMessage("notify").
						Set("content", fmt.Sprintf("%s 下载完成", m.Name)),
				)
			}

			a.delMission(gid)

		case "error":
			a.Send(
				core.NewMessage("notify").
					Set("content", fmt.Sprintf("%s 下载失败", m.Name)),
			)

			a.delMission(gid)

		case "removed":
			a.delMission(gid)
		}
	}
}

func (a *Aria2c) getGlobalDir() {
	rpcc, _ := rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, time.Second, nil)

	m, err := rpcc.GetGlobalOption()

	if err != nil {
		log.Errorf("Failed to get global dir, dose the aria2c daemon not start?: %s", err)
		time.Sleep(5 * time.Second)
		a.getGlobalDir()
		return
	}

	a.globalDir = m["dir"].(string)

	log.Debugf("aria2 global dir: %s", a.globalDir)
}

func (a *Aria2c) addMission(gid string) {
	a.Lock()
	a.missions[gid] = &Mission{
		Gid: gid,
	}
	a.Unlock()
}

func (a *Aria2c) delMission(gid string) {
	a.Lock()
	delete(a.missions, gid)
	a.Unlock()
}
