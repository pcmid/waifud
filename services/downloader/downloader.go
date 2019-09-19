package downloader

import (
	"github.com/pcmid/waifud/services"
)

type Downloader interface {
	//Ping() bool
	//Describe() string
	services.Service
	Download(url string)
}

type BaseDownloader struct{}

func (BaseDownloader) Type() string {
	return "downloader"
}
