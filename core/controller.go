package core

import "github.com/pcmid/waifud/messages"


type Controller struct {
	ms     chan messages.Message
	Server map[string]Service
}

func (c *Controller) Register(service Service) {
	if c.Server == nil {
		c.Server = make(map[string]Service)
	}

	if c.ms == nil {
		c.ms = make(chan messages.Message)
	}

	c.Server[service.Type()] = service
	service.SetMessageChan(c.ms)
}

func (c *Controller) RegisterSender(service Service) {

	service.SetMessageChan(c.ms)
}

func (c *Controller) Get(serverName string) Service {
	if server, ok := c.Server[serverName]; ok {
		return server
	}

	return nil
}
