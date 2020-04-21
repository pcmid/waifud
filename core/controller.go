package core

import (
	log "github.com/sirupsen/logrus"
)

type Controller struct {
	ms       chan Message
	Services map[string][]Service
}

func (c *Controller) Register(service Service) {
	if c.Services == nil {
		c.Services = make(map[string][]Service)
	}

	if c.ms == nil {
		c.ms = make(chan Message)
	}

	for _, t := range service.ListeningTypes() {
		if c.Services[t] == nil {
			c.Services[t] = []Service{}
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

func (c *Controller) Poll() {

	for {
		message := <-c.ms
		log.Tracef("New Message: %s: %v", message.Type, message.Message())
		rec := message.Type

		for _, service := range c.Services[rec] {
			go func(service Service) {
				service.Handle(message)
			}(service)
		}
	}
}
