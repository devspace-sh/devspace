package framework

import (
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
)

func NewDefaultFactory() factory.Factory {
	return &DefaultFactory{
		Factory: factory.DefaultFactory(),
		log:     log.GetInstance(),
	}
}

type DefaultFactory struct {
	factory.Factory

	log log.Logger
}

func (f *DefaultFactory) SetLog(log log.Logger) {
	f.log = log
}

func (f *DefaultFactory) GetLog() log.Logger {
	return log.GetInstance()
}
