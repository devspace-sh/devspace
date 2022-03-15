package hook

import (
	"context"
	runtimevar "github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
)

// RemoteHook is a hook that is executed in a container
type RemoteHook interface {
	ExecuteRemotely(hook *latest.HookConfig, podContainer *selector.SelectedPodContainer, client kubectl.Client, config config.Config, dependencies []types.Dependency, log logpkg.Logger) error
}

func NewRemoteHook(hook RemoteHook) Hook {
	return &remoteHook{
		Hook:            hook,
		WaitingStrategy: targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2),
	}
}

func NewRemoteHookWithWaitingStrategy(hook RemoteHook, waitingStrategy targetselector.WaitingStrategy) Hook {
	return &remoteHook{
		Hook:            hook,
		WaitingStrategy: waitingStrategy,
	}
}

type remoteHook struct {
	Hook            RemoteHook
	WaitingStrategy targetselector.WaitingStrategy
}

func (r *remoteHook) Execute(hook *latest.HookConfig, client kubectl.Client, config config.Config, dependencies []types.Dependency, extraEnv map[string]string, log logpkg.Logger) error {
	if client == nil {
		return errors.Errorf("Cannot execute hook '%s': kube client is not initialized", ansi.Color(hookName(hook), "white+b"))
	}

	var (
		imageSelectors []imageselector.ImageSelector
		err            error
	)
	if hook.Container.ImageSelector != "" {
		if config == nil || config.Generated() == nil {
			return errors.Errorf("Cannot execute hook '%s': config is not loaded", ansi.Color(hookName(hook), "white+b"))
		}

		if hook.Container.ImageSelector != "" {
			imageSelector, err := runtimevar.NewRuntimeResolver(true).FillRuntimeVariablesAsImageSelector(hook.Container.ImageSelector, config, dependencies)
			if err != nil {
				return err
			}

			imageSelectors = append(imageSelectors, *imageSelector)
		}
	}

	executed, err := r.execute(hook, imageSelectors, client, config, dependencies, log)
	if err != nil {
		return err
	} else if !executed {
		log.Infof("Skip hook '%s', because no running containers were found", ansi.Color(hookName(hook), "white+b"))
	}

	return nil
}

func (r *remoteHook) execute(hook *latest.HookConfig, imageSelector []imageselector.ImageSelector, client kubectl.Client, config config.Config, dependencies []types.Dependency, log logpkg.Logger) (bool, error) {
	labelSelector := ""
	if len(hook.Container.LabelSelector) > 0 {
		labelSelector = labels.Set(hook.Container.LabelSelector).String()
	}

	timeout := int64(150)
	if hook.Container.Timeout > 0 {
		timeout = hook.Container.Timeout
	}

	wait := false
	if hook.Container.Wait == nil || *hook.Container.Wait {
		log.Infof("Waiting for running containers for hook '%s'", ansi.Color(hookName(hook), "white+b"))
		wait = true
	}

	// build target selector
	targetSelectorOptions := targetselector.NewOptionsFromFlags(hook.Container.ContainerName, labelSelector, targetselector.ToStringImageSelector(imageSelector), hook.Container.Namespace, hook.Container.Pod).
		WithTimeout(timeout).
		WithWait(wait).
		WithWaitingStrategy(r.WaitingStrategy)

	// select container
	podContainer, err := targetselector.GlobalTargetSelector.SelectSingleContainer(context.TODO(), client, targetSelectorOptions, log)
	if err != nil {
		if _, ok := err.(*targetselector.NotFoundErr); ok {
			return false, nil
		}

		return false, err
	}

	// execute the hook in the container
	err = r.Hook.ExecuteRemotely(hook, podContainer, client, config, dependencies, log)
	if err != nil {
		return false, err
	}

	return true, nil
}
