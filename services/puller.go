package services

import (
	"encoding/gob"
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/pcmid/waifud/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"sync"
	"time"
)

const MinTtl = 600

const (
	Sub    int = 0x01
	UnSub      = 0x02
	GetSub     = 0x04
)

func init() {
	core.Register(&Puller{})
}

type Puller struct {
	minTTL    time.Duration
	savedPath string
	feeds     map[string]*Feed
	sync.RWMutex
	core.Receiver
	core.Sender
}

type Feed struct {
	gofeed.Feed
	Tag         string
	URL         string
	FailedCount int
}

func (p *Puller) Name() string {
	return "puller"
}

func (p *Puller) ListeningTypes() []string {
	return []string{
		"subscription",
	}
}

func (p *Puller) Start() {
	p.Init()
	p.Serve()
}

func (p *Puller) Init() {
	p.feeds = make(map[string]*Feed)
	p.minTTL = time.Duration(MinTtl)
	if viper.IsSet("service.puller.min-ttl") {
		p.minTTL = viper.GetDuration("service.puller.min-ttl")
	}
	log.Tracef("set database min ttl %d", p.minTTL)
	p.savedPath = "waifud.gob"
	if viper.IsSet("service.puller.saved-path") {
		p.savedPath = viper.GetString("service.puller.saved-path")
	}
	log.Infof("database saved as %s", p.savedPath)
	isExist := func(filename string) bool {
		_, err := os.Stat(filename)
		return err == nil || os.IsExist(err)
	}
	if isExist(p.savedPath) {
		p.restore(p.savedPath)
	}
}

func (p *Puller) Serve() {
	log.Debug("puller serve")
	tick := time.NewTicker(time.Second * p.minTTL)
	for {
		<-tick.C
		p.update()
		p.save()
	}
}

func (p *Puller) Handle(message core.Message) {
	switch message.Get("operation").(int) {
	case Sub:
		_url := message.Get("content").(string)
		if feed, ok := p.feeds[_url]; ok {
			log.Errorf("Feed %s already existed", feed.Title)
			message.Reply(core.NewMessage("response").
				Set("message", fmt.Sprintf("订阅 %s 已经存在", p.feeds[_url].Tag)).
				Set("code", -1),
			)
			return
		}
		feed := &Feed{
			URL: _url,
			Tag: message.Get("tag").(string),
		}
		if info, err := gofeed.NewParser().ParseURL(_url); err != nil {
			log.Warnf("Failed to add feed: %s", err)
			message.Reply(core.NewMessage("response").
				Set("message", "添加订阅失败").
				Set("code", -1),
			)
			return
		} else {
			feed.Title = info.Title
		}
		p.Lock()
		p.feeds[_url] = feed
		p.Unlock()

		log.Infof("Add subscribe %s successfully", _url)

		var name string
		if len(feed.Tag) != 0 {
			name = feed.Tag
		} else if len(feed.Title) != 0 {
			name = feed.Title
		} else {
			name = _url
		}

		message.Reply(core.NewMessage("response").
			Set("message", fmt.Sprintf("订阅成功: %s", name)).
			Set("code", 0),
		)

		p.update()

	case UnSub:
		_url := message.Get("content").(string)
		if feed, ok := p.feeds[_url]; ok {
			p.Lock()
			delete(p.feeds, _url)
			p.Unlock()
			message.Reply(core.NewMessage("response").
				Set("message", fmt.Sprintf("成功取消订阅: %s", feed.Title)).
				Set("code", 0),
			)
		} else {
			message.Reply(core.NewMessage("notify").
				Set("message", "订阅不存在！").
				Set("code", 0),
			)
		}

	case GetSub:
		var feeds []*Feed
		p.RLock()
		for _, feed := range p.feeds {
			feeds = append(feeds, feed)
		}
		p.RUnlock()
		message.Reply(core.NewMessage("feeds").
			Set("feeds", feeds),
		)
	}
}

func (p *Puller) save() {
	file, err := os.Create(p.savedPath)
	defer func() {
		_ = file.Close()
	}()
	if err == os.ErrExist {
		file, _ = os.Open(p.savedPath)
		defer func() {
			_ = file.Close()
		}()
	} else if err != nil {
		log.Errorf("failed to save database: %s", err)
		return
	}
	// only save the feeds
	enc := gob.NewEncoder(file)
	p.RLock()
	_ = enc.Encode(p.feeds)
	p.RUnlock()
}

func (p *Puller) restore(path string) {
	file, err := os.Open(path)
	if err != err {
		log.Errorf("failed to restore database: %s", err)
		return
	}
	dec := gob.NewDecoder(file)
	p.Lock()
	_ = dec.Decode(&p.feeds)
	p.Unlock()
}

func (p *Puller) merge(feed *Feed) (update []*gofeed.Item) {
	old := p.feeds[feed.URL]
	// new feed
	if old.Feed.PublishedParsed == nil {
		update = feed.Items
	} else if feed.PublishedParsed.After(*old.PublishedParsed) {
		for _, item := range feed.Items {
			if item.PublishedParsed.After(*old.PublishedParsed) {
				update = append(update, item)
			}
		}
		feed.Items = append(old.Items, update...)
	}
	p.feeds[feed.URL] = feed
	return
}

func (p *Puller) update() {
	log.Debug("try to update database")
	p.Lock()
	defer p.Unlock()
	for u, feed := range p.feeds {
		newData, err := gofeed.NewParser().ParseURL(u)
		if err != nil {
			log.Errorf("Failed to parse %s: %s\n", u, err.Error())
			feed.FailedCount++
			if feed.FailedCount > 5 {
				log.Errorf("%s has failed over 5 times", feed.URL)
			}
			continue
		}
		feed.FailedCount = 0
		updated := p.merge(&Feed{
			Feed: *newData,
			URL:  u,
			Tag:  feed.Tag,
		})
		for _, item := range updated {
			for _, enclosure := range item.Enclosures {
				u, _ := url.Parse(enclosure.URL)
				q := u.Query()
				u.RawQuery = q.Encode()
				resp := p.Send(core.NewMessage("item").
					Set("content", u.String()).
					Set("dir", feed.Tag),
				).WaitResponse()
				if resp.Get("code") != 0 {
					p.Send(core.NewMessage("notify").
						Set("content", resp.Get("message")),
					)
				}
			}
		}
	}
}
