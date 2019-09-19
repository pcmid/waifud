package client

import (
	"github.com/pcmid/waifud/services"
)

type Client interface {
	services.Service
	//Describe() string
	Subscribe(url string)
	UnSubscribe(url string)
}

type BaseClient struct{}


func (bc *BaseClient) Type() string {
	return "client"
}
