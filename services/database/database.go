package database

import (
	"encoding/gob"
	"github.com/mmcdole/gofeed"
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/services"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"time"
)

const MIN_TTL = 600

const (
	AddFeed int = 0x01
	DelFeed     = 0x02
)

func init() {
	services.ServiceMap["database"] = &Database{}
}

type Database struct {
	rms chan messages.Message
	sms chan messages.Message

	Feeds map[string]*Feed

	minTTL    time.Duration
	savedPath string
}

func (db *Database) Types() []string {
	return []string{db.Type()}
}

func (db *Database) Init() {
	//panic("implement me")
	db.Feeds = make(map[string]*Feed)
	db.rms = make(chan messages.Message)

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


	isExist := func (filename string) bool {
		_, err := os.Stat(filename)
		return err == nil || os.IsExist(err)
	}

	if isExist(db.savedPath) {
		db.restore(db.savedPath)
	}
}

type Feed struct {
	gofeed.Feed

	URL        string
	FiledCount int
}

func (db *Database) Name() string {
	return "database"
}

func (db *Database) Serve() {
	//db := new(Database)

	log.Debug("database serve")

	db.Poll()
}

func (db *Database) Poll() {

	tick := time.NewTicker(time.Second * db.minTTL)
	for {

		select {
		case <-tick.C:
			db.Update()

		case m1 := <-db.rms:

			m := m1.(*messages.DBMessage)

			//res := &messages.ResultMessage{}

			switch m.Code {
			case AddFeed:
				if feed, ok := db.Feeds[m.URL]; ok {
					//_, _ = fmt.Sscanf(res.M, "Feed %s already exsits\n", feed.Title)
					log.Errorf("Feed %s already exsits", feed.Title)
					continue
				}

				db.Feeds[m.URL] = &Feed{}
				//_, _ = fmt.Sscanf(res.M, "Add subscribe %s successf", db.Feeds[m.URL].Title)
				log.Infof("Add subscribe %s successfully", m.URL)

				db.Update()
			case DelFeed:
				delete(db.Feeds, m.URL)
			}

			//db.Send(res)
		}
		db.save()

	}

}

func (db *Database) Merge(feed *Feed) (update []*gofeed.Item) {

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

func (db *Database) Update() {

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

		updated := db.Merge(&Feed{
			Feed: *newData,
			URL:  u,
		})

		//log.Trace(feed, updated)

		for _, item := range updated {
			for _, enclosure := range item.Enclosures {
				u, _ := url.Parse(enclosure.URL)
				q := u.Query()
				u.RawQuery = q.Encode()

				db.Send(&messages.DLMessage{URL: u.String()})
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
	} else if err != nil  {
		log.Errorf("failed to save database: %s", err)
		return
	}

	// only save the feeds
	enc := gob.NewEncoder(file)
	_ = enc.Encode(db.Feeds)
}

func (db *Database)restore(path string) {

	file, err := os.Open(path)

	if err != err {
		log.Errorf("failed to restore database: %s", err)
		return
	}

	dec := gob.NewDecoder(file)
	_ = dec.Decode(&db.Feeds)
}
