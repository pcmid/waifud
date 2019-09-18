package main

import (
	"github.com/pcmid/waifud/core"
	"github.com/pcmid/waifud/service/client"
	"github.com/pcmid/waifud/service/database"
	"github.com/pcmid/waifud/service/downloader"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {

	logFormatter := new(log.TextFormatter)
	logFormatter.FullTimestamp = true
	logFormatter.TimestampFormat = "2006-01-02 15:04:05"

	logLevel := os.Getenv("LOGLEVEL")

	levelMap := map[string]log.Level{
		"TRACE": log.TraceLevel,
		"DEBUG": log.DebugLevel,

		"INFO":  log.InfoLevel,
		"WARN":  log.WarnLevel,
		"ERROR": log.ErrorLevel,

		"FATAL": log.FatalLevel,
		"PANIC": log.PanicLevel,
	}

	log.SetLevel(log.InfoLevel)

	if level, ok := levelMap[logLevel]; ok {
		log.SetLevel(level)
	}

	log.SetFormatter(logFormatter)
}

func main() {

	db := &database.Database{}
	//curl := &downloader.Curl{}
	//jc := &client.JsonClient{}
	//id := &downloader.InBuilt{}

	aria2 := &downloader.Aria2c{}
	telebot := &client.TeleBot{}
	c := &core.Controller{}

	c.Register(db)
	c.Register(aria2)
	c.Register(telebot)

	c.Poll()

	//ms <- &messages.DBMessage{
	//
	//	Code: service.AddFeed,
	//
	//	URL: "https://bangumi.moe/rss/tags/548ee0ea4ab7379536f56354+548ee2ce4ab7379536f56358",
	//}
	//
	//time.Sleep(10 * time.Second)
	//
	//ms <- &messages.DBMessage{
	//
	//	Code: service.AddFeed,
	//
	//	URL: "https://bangumi.moe/rss/latest",
	//}

	select {}
}
