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

	feeds map[string]*Feed

	rms chan core.Message
	sms chan core.Message
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

func (p *Puller) Init() {
	p.feeds = make(map[string]*Feed)
	p.rms = make(chan core.Message)

	p.minTTL = time.Duration(MinTtl)

	if viper.IsSet("service.database.min-ttl") {
		p.minTTL = viper.GetDuration("service.database.min-ttl")
	}

	log.Tracef("set database min ttl %d", p.minTTL)

	p.savedPath = "waifud.gob"
	if viper.IsSet("service.database.saved-path") {
		p.savedPath = viper.GetString("service.database.saved-path")
	}

	log.Infof("database saved as  %s", p.savedPath)

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

	p.serve()
}

func (p *Puller) Handle(message core.Message) {
	p.rms <- message
}

func (p *Puller) SetMessageChan(sms chan core.Message) {
	p.sms = sms
}

func (p *Puller) Send(message core.Message) {
	if p.sms == nil {
		return
	}
	p.sms <- message
}

func (p *Puller) serve() {

	tick := time.NewTicker(time.Second * p.minTTL)
	for {

		select {
		case <-tick.C:
			p.update()

		case m := <-p.rms:

			switch m.Get("operation").(int) {
			case Sub:
				_url := m.Get("content").(string)

				if feed, ok := p.feeds[_url]; ok {
					log.Errorf("Feed %s already existed", feed.Title)
					p.Send(
						core.NewMessage("notify").
							Set("content", fmt.Sprintf("订阅 %s 已经存在", p.feeds[_url].Title)),
					)

					continue
				}

				var title string
				if feed, err := gofeed.NewParser().ParseURL(_url); err != nil {
					log.Warnf("Failed to add feed: %s", err)
					p.Send(
						core.NewMessage("notify").
							Set("content", "订阅失败"),
					)
					continue
				} else {
					title = feed.Title
				}

				p.feeds[_url] = &Feed{
					URL: _url,
					dir: m.Get("dir").(string),
				}
				log.Infof("Add subscribe %s successfully", _url)

				p.Send(
					core.NewMessage("notify").
						Set("content", fmt.Sprintf("订阅成功: %s", title)),
				)

				p.update()

			case UnSub:
				_url := m.Get("content").(string)

				if feed, ok := p.feeds[_url]; ok {
					delete(p.feeds, _url)

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
				for _, feed := range p.feeds {
					feeds = append(feeds, feed)
				}

				m := core.NewMessage("feeds").
					Set("feeds", feeds)

				p.Send(m)
			}
		}
		p.save()
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
	_ = enc.Encode(p.feeds)
}

func (p *Puller) restore(path string) {

	file, err := os.Open(path)

	if err != err {
		log.Errorf("failed to restore database: %s", err)
		return
	}

	dec := gob.NewDecoder(file)
	_ = dec.Decode(&p.feeds)
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
