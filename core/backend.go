package core

import (
	"github.com/pcmid/waifud/messages"
	log "github.com/sirupsen/logrus"
)

func init() {
}

func (c *Controller) Poll() chan messages.Message {

	for _, ss := range c.Services {

		for _, s := range ss {
			s.Init()
			go func(service Service) {
				log.Infof("Service %s Start...", service.Name())
				service.Serve()
			}(s)
		}

	}

	go func() {
		for {
			message := <-c.ms
			log.Tracef("New Message to %s: %v", message.Reciver(), message.Message())
			services := c.Get(message.Reciver())

			if services == nil {
				log.Errorf("Failed to get recv for rms %s: %s: %v", message.Reciver(), message.Describe(), message.Message())
				continue
			}

			for _, service := range services {
				go func(s Service) {
					s.Handle(message)
				}(service)
			}
		}
	}()

	return c.ms
}
