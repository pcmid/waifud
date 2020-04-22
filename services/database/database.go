package database

import (
	"encoding/gob"
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/pcmid/waifud/core"
	"github.com/pcmid/waifud/services"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"time"
)

const MIN_TTL = 600

const (
	Sub    int = 0x01
	UnSub      = 0x02
	GetSub     = 0x04
)

func init() {
	services.ServiceMap["database"] = &Database{}
}

type Database struct {
	rms chan *Subscription
	sms chan core.Message

	Feeds map[string]*Feed

	minTTL    time.Duration
	savedPath string
}

type Feed struct {
	gofeed.Feed

	URL        string
	FiledCount int
}

type Subscription struct {
	Op  int
	Url string
}

func (db *Database) ListeningTypes() []string {
	return []string{
		"subscription",
	}
}

func (db *Database) Init() {
	//panic("implement me")
	db.Feeds = make(map[string]*Feed)
	db.rms = make(chan *Subscription)

	db.minTTL = time.Duration(MIN_TTL)

	if viper.IsSet("service.Database.min-ttl") {
		db.minTTL = viper.GetDuration("service.Database.min-ttl")
	}

	log.Tracef("set database min ttl %d", db.minTTL)

	db.savedPath = "waifud.gob"
	if viper.IsSet("service.Database.saved-path") {
		db.savedPath = viper.GetString("service.Database.saved-path")
	}

	log.Infof("database saved as  %s", db.savedPath)

	isExist := func(filename string) bool {
		_, err := os.Stat(filename)
		return err == nil || os.IsExist(err)
	}

	if isExist(db.savedPath) {
		db.restore(db.savedPath)
	}
}

func (db *Database) Name() string {
	return "database"
}

func (db *Database) Serve() {
	log.Debug("database serve")

	db.Poll()
}

func (db *Database) Handle(message core.Message) {
	db.rms <- message.Message().(*Subscription)
}

func (db *Database) SetMessageChan(sms chan core.Message) {
	db.sms = sms
}

func (db *Database) Send(message core.Message) {
	if db.sms == nil {
		return
	}
	db.sms <- message
}

func (db *Database) Poll() {

	tick := time.NewTicker(time.Second * db.minTTL)
	for {

		select {
		case <-tick.C:
			db.update()

		case m := <-db.rms:
			switch m.Op {
			case Sub:
				if feed, ok := db.Feeds[m.Url]; ok {
					log.Errorf("Feed %s already exsits", feed.Title)
					db.Send(core.Message{
						Type: "notify",
						Msg:  fmt.Sprintf("订阅 %s 已经存在", db.Feeds[m.Url].Title),
					})
					continue
				}

				db.Feeds[m.Url] = &Feed{}
				log.Infof("Add subscribe %s successfully", m.Url)
				db.update()

				db.Send(core.Message{
					Type: "notify",
					Msg:  fmt.Sprintf("订阅成功: %s", db.Feeds[m.Url].Title),
				})
			case UnSub:
				if feed, ok := db.Feeds[m.Url]; ok {
					delete(db.Feeds, m.Url)

					db.Send(core.Message{
						Type: "notify",
						Msg:  fmt.Sprintf("成功取消订阅: %s", feed.Title),
					})
				} else {
					db.Send(core.Message{
						Type: "notify",
						Msg:  "订阅不存在！",
					})
				}

			case GetSub:
				var feeds []*Feed
				for _, f := range db.Feeds {
					feeds = append(feeds, f)
				}
				db.Send(core.Message{
					Type: "feeds",
					Msg:  db.Feeds,
				})
			}
		}
		db.save()

	}

}

func (db *Database) save() {

	file, err := os.Create(db.savedPath)
	defer func() {
		_ = file.Close()
	}()

	if err == os.ErrExist {
		file, _ = os.Open(db.savedPath)
		defer func() {
			_ = file.Close()
		}()
	} else if err != nil {
		log.Errorf("failed to save database: %s", err)
		return
	}

	// only save the feeds
	enc := gob.NewEncoder(file)
	_ = enc.Encode(db.Feeds)
}

func (db *Database) restore(path string) {

	file, err := os.Open(path)

	if err != err {
		log.Errorf("failed to restore database: %s", err)
		return
	}

	dec := gob.NewDecoder(file)
	_ = dec.Decode(&db.Feeds)
}

func (db *Database) merge(feed *Feed) (update []*gofeed.Item) {

	old := db.Feeds[feed.URL]

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

	db.Feeds[feed.URL] = feed

	return
}

func (db *Database) update() {

	log.Debug("try to update database")

	for u, feed := range db.Feeds {
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

		updated := db.merge(&Feed{
			Feed: *newData,
			URL:  u,
		})

		for _, item := range updated {
			for _, enclosure := range item.Enclosures {
				u, _ := url.Parse(enclosure.URL)
				q := u.Query()
				u.RawQuery = q.Encode()

				db.Send(core.Message{
					Type: "enclosure",
					Msg:  u.String(),
				})
			}
		}
	}
}
