package services

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/pcmid/waifud/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/zyxar/argo/rpc"
	"net/url"
	"os"
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

	session string

	globalDir string
	missions  map[string]*Mission

	rpcc rpc.Client

	sync.Mutex

	core.Receiver
	core.Sender
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
		"link",
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

	if viper.IsSet("service.aria2c.session") {
		a.session = viper.GetString("service.aria2c.session")
		log.Tracef("set aria2c session: %s", a.session)
	} else {
		log.Warn("aria2c session not found, session will not be saved")
	}

	a.restore()

	for err := a.connect(); err != nil; {
		log.Errorf("Failed to connect aria2 jsonrpc: %s", err)

		// reconnect
		time.Sleep(10 * time.Second)
		err = a.connect()
	}

	a.getGlobalDir()
}

func (a *Aria2c) Serve() {
	tick := time.NewTicker(10 * time.Second)
	tickForSave := time.NewTicker(5 * time.Minute)

	for {
		select {
		case <-tick.C:
			a.update()
			a.check()
		case <-tickForSave.C:
			a.save()
		}
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

			a.Reply(message, core.NewMessage("status").
				Set("missions", missions),
			)

		}
	case "item":
		uri := message.Get("content").(string)
		dir := message.Get("dir").(string)
		if err := a.download(uri, dir); err != nil {
			log.Errorf("Failed to AddURL for aria2c: %s", err)
			a.Send(
				core.NewMessage("notify").
					Set("content", "下载更新失败"),
			)
		}

		// slow down for next
		time.Sleep(10 * time.Millisecond)
	case "link":
		uri := message.Get("url").(string)
		dir := message.Get("dir").(string)
		if err := a.download(uri, dir); err != nil {
			log.Errorf("Failed to AddURL for aria2c: %s", err)
			a.Send(
				core.NewMessage("notify").
					Set("content", "添加下载失败"),
			)
		} else {
			a.Send(
				core.NewMessage("notify").
					Set("content", "新的任务已添加"),
			)
		}
	}
}

func (a *Aria2c) connect() (err error) {
	if a.rpcc, err = rpc.New(context.Background(), a.rpcUrl, a.rpcSecret, 10*time.Second, nil); err != nil {
		log.Debugf("Failed to create aria2 connection: %s", err)
		return
	}

	//for test
	_, err = a.rpcc.GetSessionInfo()

	return
}

func (a *Aria2c) download(url, dir string) error {
	log.Infof("download %s at %s", url, a.globalDir+"/"+dir)

	gid, err := a.rpcc.AddURI(url, rpc.Option{
		"dir": a.globalDir + "/" + dir,
	})

	if err != nil {
		return err
	}

	a.addMission(gid)

	return nil
}

func (a *Aria2c) update() {
	for gid, mission := range a.missions {
		s, err := a.rpcc.TellStatus(gid)

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
			file, _ := url.ParseRequestURI(s.Files[0].URIs[0].URI)
			mission.Name = file.Path[strings.LastIndex(file.Path, "/")+1:]
		} else if s.BitTorrent.Info.Name != "" {
			// from torrent
			mission.Name = s.BitTorrent.Info.Name
		} else {
			// from torrent link
			mission.Name = "[METADATA]" + s.InfoHash
		}

		if mission.Name == "" || mission.Status == "" {
			log.Warnf("bug: %s", mission)
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
		time.Sleep(10 * time.Millisecond)
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

func (a *Aria2c) save() {
	if a.session == "" {
		return
	}

	file, err := os.Create(a.session)
	defer func() {
		_ = file.Close()
	}()

	if err == os.ErrExist {
		file, _ = os.Open(a.session)
		defer func() {
			_ = file.Close()
		}()
	} else if err != nil {
		log.Errorf("Failed to open aria2c session: %s", err)
		return
	}

	enc := gob.NewEncoder(file)

	a.Lock()
	_ = enc.Encode(a.missions)
	a.Unlock()
}

func (a *Aria2c) restore() {
	if a.session == "" {
		return
	}

	file, err := os.Open(a.session)

	if err != err {
		log.Errorf("Failed to restore session: %s", err)
		return
	}

	dec := gob.NewDecoder(file)
	_ = dec.Decode(&a.missions)
}

func (a *Aria2c) getGlobalDir() {

	var m rpc.Option
	var err error

	for m, err = a.rpcc.GetGlobalOption(); err != nil; {
		log.Errorf("Failed to get global dir: %s", err)
		m, err = a.rpcc.GetGlobalOption()
		time.Sleep(5 * time.Second)
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
