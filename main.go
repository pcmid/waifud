package main

import (
	"fmt"
	"github.com/pcmid/waifud/core"
	_ "github.com/pcmid/waifud/services"
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

//  go build -ldflags "-X main.version=version"
var version = ""

var cliConfFile = flag.StringP("config", "c", "config.yaml", "config file")
var cliHelp = flag.BoolP("help", "h", false, "print this help")
var cliVersion = flag.BoolP("version", "v", false, "print waifud version")

func main() {

	flag.Parse()

	if *cliHelp {
		flag.PrintDefaults()
		return
	}

	if *cliVersion {
		fmt.Printf("waidud %s\n", version)
		return
	}

	viper.SetConfigType("yaml")
	viper.SetConfigFile(*cliConfFile)
	err := viper.ReadInConfig()

	if err != nil {
		log.Fatalf("Fatal error config file: %s", err)
	}

	c := &core.Scheduler{}

	for service := range viper.GetStringMapStringSlice("service") {
		c.Init(service)
		log.Tracef("Init %s", service)
	}

	c.Loop()
}
