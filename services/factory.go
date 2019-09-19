package services

import (
	log "github.com/sirupsen/logrus"
)

var ServiceMap map[string]Service

func init()  {
	ServiceMap = make(map[string]Service)
}

func Get(name string) Service {
	if s, ok := ServiceMap[name]; ok {
		return s
	}

	log.Errorf("service %s not found", name)
	return nil
}
