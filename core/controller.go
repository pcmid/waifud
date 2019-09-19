package core

import (
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/services"
)


type Controller struct {
	ms       chan messages.Message
	Services map[string][]services.Service
}

func (c *Controller) Register(service services.Service) {
	if c.Services == nil {
		c.Services = make(map[string][]services.Service)
	}

	if c.ms == nil {
		c.ms = make(chan messages.Message)
	}

	if c.Services[service.Type()] == nil {
		c.Services[service.Type()] = []services.Service{}
	}

	c.Services[service.Type()] = append(c.Services[service.Type()], service)
	service.SetMessageChan(c.ms)
}

func (c *Controller) RegisterSender(service services.Service) {

	service.SetMessageChan(c.ms)
}

func (c *Controller) Get(serverName string) []services.Service {
	if server, ok := c.Services[serverName]; ok {
		return server
	}

	return nil
}
