package core

import (
	log "github.com/sirupsen/logrus"
)

type Scheduler struct {
	messages chan *Message
	services map[string][]Service
}

func (s *Scheduler) Init(serviceName string) {

	var service Service
	var ok bool

	if service, ok = Services[serviceName]; !ok {
		log.Fatalf("service %s not found", serviceName)
	}

	if s.services == nil {
		s.services = make(map[string][]Service)
	}

	if s.messages == nil {
		s.messages = make(chan *Message)
	}

	for _, t := range service.ListeningTypes() {
		if s.services[t] == nil {
			s.services[t] = []Service{}
		}
		s.services[t] = append(s.services[t], service)
	}

	service.SetMessageChan(s.messages)

	go func() {
		service.Init()
		log.Infof("Service %s Start...", service.Name())
		service.Serve()
	}()
}

func (s *Scheduler) Loop() {

	for {
		message := <-s.messages
		log.Tracef("New Message: %s: %v", message.Type, message)
		rec := message.Type

		for _, service := range s.services[rec] {
			go func(service Service) {
				service.Handle(message)
			}(service)
		}
	}
}
