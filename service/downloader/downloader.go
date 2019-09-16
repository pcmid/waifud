package downloader

import (
	"github.com/pcmid/waifud/core"
)

type Downloader interface {
	//Ping() bool
	//Describe() string
	core.Service
	Download(url string)
}

type BaseDownloader struct{}

func (BaseDownloader) Type() string {
	return "downloader"
}
