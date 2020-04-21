package services

import (
	"github.com/pcmid/waifud/core"
	log "github.com/sirupsen/logrus"
)

var ServiceMap map[string]core.Service

func init()  {
	ServiceMap = make(map[string]core.Service)
}

func Get(name string) core.Service {
	if s, ok := ServiceMap[name]; ok {
		return s
	}

	log.Errorf("service %s not found", name)
	return nil
}
