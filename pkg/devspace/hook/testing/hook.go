package hook

import(
	"github.com/devspace-cloud/devspace/pkg/devspace/hook"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

// FakeHook is a fake implementation of the interface Hook
type FakeHook struct{}

// Execute is a fake implementation of this function
func (f *FakeHook) Execute(when hook.When, stage hook.Stage, which string, log log.Logger) error {
	return nil
}