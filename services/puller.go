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

	sync.RWMutex
	feeds map[string]*Feed

	core.Receiver
	core.Sender
}

type Feed struct {
	gofeed.Feed

	URL        string
	FiledCount int

	dir string
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

	if viper.IsSet("service.database.min-ttl") {
		p.minTTL = viper.GetDuration("service.database.min-ttl")
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
	log.Debug("database serve")
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
			p.Send(
				core.NewMessage("notify").
					Set("content", fmt.Sprintf("订阅 %s 已经存在", p.feeds[_url].Title)),
			)

			return
		}

		var title string
		if feed, err := gofeed.NewParser().ParseURL(_url); err != nil {
			log.Warnf("Failed to add feed: %s", err)
			p.Send(
				core.NewMessage("notify").
					Set("content", "订阅失败"),
			)
			return
		} else {
			title = feed.Title
		}

		p.Lock()
		p.feeds[_url] = &Feed{
			URL: _url,
			dir: message.Get("dir").(string),
		}
		p.Lock()
		log.Infof("Add subscribe %s successfully", _url)

		p.Send(
			core.NewMessage("notify").
				Set("content", fmt.Sprintf("订阅成功: %s", title)),
		)

		p.update()

	case UnSub:
		_url := message.Get("content").(string)

		if feed, ok := p.feeds[_url]; ok {
			p.Lock()
			delete(p.feeds, _url)
			p.Unlock()

			p.Send(
				core.NewMessage("notify").
					Set("content", fmt.Sprintf("成功取消订阅: %s", feed.Title)),
			)

		} else {
			p.Send(
				core.NewMessage("notify").
					Set("content", "订阅不存在！"),
			)
		}

	case GetSub:
		var feeds []*Feed
		p.RLock()
		for _, feed := range p.feeds {
			feeds = append(feeds, feed)
		}
		p.RUnlock()

		p.Reply(message, core.NewMessage("feeds").
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

			feed.FiledCount++
			if feed.FiledCount > 5 {
				log.Errorf("%s has failed over 5 times", feed.URL)
			}
			continue
		}

		feed.FiledCount = 0

		updated := p.merge(&Feed{
			Feed: *newData,
			URL:  u,
		})

		for _, item := range updated {
			for _, enclosure := range item.Enclosures {
				u, _ := url.Parse(enclosure.URL)
				q := u.Query()
				u.RawQuery = q.Encode()

				p.Send(
					core.NewMessage("item").
						Set("content", u.String()).
						Set("dir", feed.dir),
				)
			}
		}
	}
}
