package hook

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/hook"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// FakeHook is a fake implementation of the interface Hook
type FakeHook struct{}

func (f *FakeHook) Execute(when hook.When, stage hook.Stage, which string, context hook.Context, log log.Logger) error {
	return nil
}

func (f *FakeHook) ExecuteMultiple(when hook.When, stage hook.Stage, whichs []string, context hook.Context, log log.Logger) error {
	return nil
}

func (f *FakeHook) OnError(stage hook.Stage, whichs []string, context hook.Context, log log.Logger) {
	return
}
