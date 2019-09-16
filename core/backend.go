package core

import (
	"github.com/pcmid/waifud/messages"
	log "github.com/sirupsen/logrus"
)

func init() {
}

func (c *Controller) Poll() chan messages.Message {

	for _, s := range c.Server {
		s.Init()
		go func(service Service) {
			log.Infof("Service %s Start...", service.Name())
			service.Serve()
		}(s)
	}

	go func() {
		for {
			message := <-c.ms
			log.Tracef("New Message to %s :%v", message.Reciver(), message.Message())
			rec := c.Get(message.Reciver())

			if rec == nil {
				log.Errorf("Failed to get recv for rms %s:%s:%v", message.Reciver(), message.Describe(), message.Message())
				continue
			}
			go rec.Handle(message)
		}
	}()

	return c.ms
}
