package client

import "github.com/pcmid/waifud/core"

type Client interface {
	core.Service
	//Describe() string
	Subscribe(url string)
	UnSubscribe(url string)
}

type BaseClient struct{}


func (bc *BaseClient) Type() string {
	return "client"
}
