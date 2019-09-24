package main

import (
	"github.com/pcmid/waifud/core"
	"github.com/pcmid/waifud/services"
	_ "github.com/pcmid/waifud/services/client"
	_ "github.com/pcmid/waifud/services/database"
	_ "github.com/pcmid/waifud/services/downloader"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
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

var confFile = flag.StringP("config","c","config.toml", "config file")

func main() {

	flag.Parse()

	viper.SetConfigFile(*confFile)
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("Fatal error config file: %s", err)
	}

	c := &core.Controller{}

	for serviceName := range viper.GetStringMapStringSlice("service") {
		c.Register(services.Get(serviceName))
		log.Tracef("Registered %s", serviceName)
	}

	c.Poll()
}
