package core

import (
	"github.com/pcmid/waifud/messages"
	"github.com/pcmid/waifud/services"
	log "github.com/sirupsen/logrus"
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

	for _, t := range service.Types() {
		if c.Services[t] == nil {
			c.Services[t] = []services.Service{}
		}
		c.Services[t] = append(c.Services[t], service)
	}

	service.SetMessageChan(c.ms)


	go func() {
		service.Init()

		log.Infof("Service %s Start...", service.Name())
		service.Serve()
	}()
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
