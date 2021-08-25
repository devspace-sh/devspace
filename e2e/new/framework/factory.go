package framework

import (
	"github.com/loft-sh/devspace/pkg/util/factory"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
)

var _ factory.Factory = &DefaultFactory{}

func NewDefaultFactory() *DefaultFactory {
	return &DefaultFactory{
		Factory: factory.DefaultFactory(),
		log: &DefaultLog{
			Logger: log.GetInstance(),
		},
	}
}

type DefaultFactory struct {
	factory.Factory

	log log.Logger
}

func (f *DefaultFactory) SetAnswerFunc(fn QuestionFn) {
	f.log.(*DefaultLog).SetQuestionFn(fn)
}

func (f *DefaultFactory) SetLog(log log.Logger) {
	f.log = log
}

func (f *DefaultFactory) GetLog() log.Logger {
	return f.log
}

type QuestionFn func(params *survey.QuestionOptions) (string, error)

type DefaultLog struct {
	log.Logger

	questionFn QuestionFn
}

func (d *DefaultLog) SetQuestionFn(fn QuestionFn) {
	d.questionFn = fn
}

func (d *DefaultLog) Question(params *survey.QuestionOptions) (string, error) {
	return d.questionFn(params)
}
