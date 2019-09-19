package database

import (
	"github.com/mmcdole/gofeed"
	"github.com/pcmid/waifud/messages"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

const MIN_TTL = 600

const (
	AddFeed int = 0x01
	DelFeed     = 0x02
)

type Database struct {
	rms chan messages.Message
	sms chan messages.Message

	feeds map[string]*Feed
}

func (db *Database) Init() {
	//panic("implement me")
	db.feeds = make(map[string]*Feed)
	db.rms = make(chan messages.Message)
}

type Feed struct {
	gofeed.Feed

	URL        string
	FiledCount int
}

func (db *Database) Name() string {
	return "Database"
}

func (db *Database) Serve() {
	//db := new(Database)

	log.Debug("database serve")

	db.Poll()
}

func (db *Database) Poll() {

	minTtl := time.Duration(MIN_TTL)

	if viper.IsSet("services.Database.MinTtl") {
		minTtl = viper.GetDuration("services.Database.MinTtl")
	}

	tick := time.NewTicker(time.Second * minTtl)
	for {

		select {
		case <-tick.C:
			db.Update()

		case m1 := <-db.rms:

			m := m1.(*messages.DBMessage)

			//res := &messages.ResultMessage{}

			switch m.Code {
			case AddFeed:
				if feed, ok := db.feeds[m.URL]; ok {
					//_, _ = fmt.Sscanf(res.M, "Feed %s already exsits\n", feed.Title)
					log.Errorf("Feed %s already exsits", feed.Title)
					continue
				}

				db.feeds[m.URL] = &Feed{}
				//_, _ = fmt.Sscanf(res.M, "Add subscribe %s successf", db.feeds[m.URL].Title)
				log.Infof("Add subscribe %s successfully", m.URL)

				db.Update()
			case DelFeed:
				delete(db.feeds, m.URL)
			}

			//db.Send(res)
		}

	}

}

func (db *Database) Merge(feed *Feed) (update []*gofeed.Item) {

	old := db.feeds[feed.URL]

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

	db.feeds[feed.URL] = feed

	return
}

func (db *Database) Update() {

	log.Debug("try to update database")

	for url, feed := range db.feeds {
		newData, err := gofeed.NewParser().ParseURL(url)
		if err != nil {
			log.Errorf("Failed to parse %s: %s\n", url, err.Error())

			feed.FiledCount++
			if feed.FiledCount > 5 {
				log.Errorf("%s has failed over 5 times", feed.URL)
			}
			continue
		}

		feed.FiledCount = 0

		updated := db.Merge(&Feed{
			Feed: *newData,
			URL:  url,
		})

		//log.Trace(feed, updated)

		for _, item := range updated {
			for _, url := range item.Enclosures {
				db.Send(&messages.DLMessage{URL: url.URL})
			}
		}
	}
}

func (db *Database) Type() string {
	return "database"
}

func (db *Database) Handle(message messages.Message) {
	db.rms <- (message).(*messages.DBMessage)
}

func (db *Database) SetMessageChan(sms chan messages.Message) {
	db.sms = sms
}

func (db *Database) Send(message messages.Message) {
	if db.sms == nil {
		return
	}
	db.sms <- message
}
