package hook

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"time"
)

// RemoteHook is a hook that is executed in a container
type RemoteHook interface {
	ExecuteRemotely(ctx Context, hook *latest.HookConfig, podContainer *kubectl.SelectedPodContainer, log logpkg.Logger) error
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

func (r *remoteHook) Execute(ctx Context, hook *latest.HookConfig, log logpkg.Logger) error {
	if ctx.Client == nil {
		return errors.Errorf("Cannot execute hook '%s': kube client is not initialized", ansi.Color(hookName(hook), "white+b"))
	}

	var imageSelector []string
	if hook.Where.Container.ImageName != "" {
		if ctx.Config == nil || ctx.Cache == nil {
			return errors.Errorf("Cannot execute hook '%s': config is not loaded", ansi.Color(hookName(hook), "white+b"))
		}

		imageSelector = targetselector.ImageSelectorFromConfig(hook.Where.Container.ImageName, ctx.Config, ctx.Cache)
	}

	executed, err := r.execute(ctx, hook, imageSelector, log)
	if err != nil {
		return err
	} else if executed == false {
		log.Infof("Skip hook '%s', because no running containers were found", ansi.Color(hookName(hook), "white+b"))
	}

	return nil
}

func (r *remoteHook) execute(ctx Context, hook *latest.HookConfig, imageSelector []string, log logpkg.Logger) (bool, error) {
	labelSelector := ""
	if len(hook.Where.Container.LabelSelector) > 0 {
		labelSelector = labels.Set(hook.Where.Container.LabelSelector).String()
	}

	timeout := int64(150)
	if hook.Where.Container.Timeout > 0 {
		timeout = hook.Where.Container.Timeout
	}

	wait := false
	if hook.Where.Container.Wait == nil || *hook.Where.Container.Wait == true {
		log.Infof("Waiting for running containers for hook '%s'", ansi.Color(hookName(hook), "white+b"))
		wait = true
	}

	// select the container
	targetSelector := targetselector.NewTargetSelector(ctx.Client)
	podContainer, err := targetSelector.SelectSingleContainer(context.TODO(), targetselector.Options{
		Selector: kubectl.Selector{
			ImageSelector: imageSelector,
			LabelSelector: labelSelector,
			Pod:           hook.Where.Container.Pod,
			ContainerName: hook.Where.Container.ContainerName,
			Namespace:     hook.Where.Container.Namespace,
		},
		Wait:            &wait,
		Timeout:         timeout,
		SortPods:        kubectl.SortPodsByNewest,
		SortContainers:  kubectl.SortContainersByNewest,
		WaitingStrategy: r.WaitingStrategy,
	}, log)
	if err != nil {
		if _, ok := err.(*targetselector.NotFoundErr); ok {
			return false, nil
		}

		return false, err
	}

	// execute the hook in the container
	log.Infof("Execute hook '%s' in container '%s/%s/%s'", ansi.Color(hookName(hook), "white+b"), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name)
	err = r.Hook.ExecuteRemotely(ctx, hook, podContainer, log)
	if err != nil {
		return false, err
	}

	return true, nil
}
