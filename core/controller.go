package core

import "github.com/pcmid/waifud/messages"


type Controller struct {
	ms       chan messages.Message
	Services map[string][]Service
}

func (c *Controller) Register(service Service) {
	if c.Services == nil {
		c.Services = make(map[string][]Service)
	}

	if c.ms == nil {
		c.ms = make(chan messages.Message)
	}

	if c.Services[service.Type()] == nil {
		c.Services[service.Type()] = []Service{}
	}

	c.Services[service.Type()] = append(c.Services[service.Type()], service)
	service.SetMessageChan(c.ms)
}

func (c *Controller) RegisterSender(service Service) {

	service.SetMessageChan(c.ms)
}

func (c *Controller) Get(serverName string) []Service {
	if server, ok := c.Services[serverName]; ok {
		return server
	}

	return nil
}
