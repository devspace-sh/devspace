package hook

import (
	"context"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

func NewWaitHook() Hook {
	return &waitHook{}
}

type waitHook struct {
	printWarning sync.Once
}

func (r *waitHook) Execute(ctx devspacecontext.Context, hook *latest.HookConfig, extraEnv map[string]string) error {
	if ctx.KubeClient() == nil {
		return errors.Errorf("Cannot execute hook '%s': kube client is not initialized", ansi.Color(hookName(hook), "white+b"))
	}

	var (
		imageSelectors []imageselector.ImageSelector
		err            error
	)
	if hook.Container.ImageSelector != "" {
		if ctx.Config() == nil || ctx.Config().LocalCache() == nil {
			return errors.Errorf("Cannot execute hook '%s': config is not loaded", ansi.Color(hookName(hook), "white+b"))
		}

		if hook.Container.ImageSelector != "" {
			imageSelector, err := runtime.NewRuntimeResolver(ctx.WorkingDir(), true).FillRuntimeVariablesAsImageSelector(ctx.Context(), hook.Container.ImageSelector, ctx.Config(), ctx.Dependencies())
			if err != nil {
				return err
			}

			imageSelectors = append(imageSelectors, *imageSelector)
		}
	}

	err = r.execute(ctx.Context(), hook, ctx.KubeClient(), imageSelectors, ctx.Log())
	if err != nil {
		return err
	}

	ctx.Log().Donef("Hook '%s' successfully executed", ansi.Color(hookName(hook), "white+b"))
	return nil
}

func (r *waitHook) execute(ctx context.Context, hook *latest.HookConfig, client kubectl.Client, imageSelector []imageselector.ImageSelector, log logpkg.Logger) error {
	labelSelector := ""
	if len(hook.Container.LabelSelector) > 0 {
		labelSelector = labels.Set(hook.Container.LabelSelector).String()
	}

	timeout := int64(150)
	if hook.Wait.Timeout > 0 {
		timeout = hook.Wait.Timeout
	}

	// wait until the defined condition will be true, this will wait initially 2 seconds
	err := wait.Poll(time.Second*2, time.Duration(timeout)*time.Second, func() (done bool, err error) {
		podContainers, err := selector.NewFilter(client).SelectContainers(ctx, selector.Selector{
			ImageSelector:   targetselector.ToStringImageSelector(imageSelector),
			LabelSelector:   labelSelector,
			Pod:             hook.Container.Pod,
			ContainerName:   hook.Container.ContainerName,
			Namespace:       hook.Container.Namespace,
			FilterContainer: selector.FilterTerminatingContainers,
		})
		if err != nil {
			return false, err
		}

		// lets check if all containers satisfy the condition
		for _, pc := range podContainers {
			if targetselector.HasPodProblem(pc.Pod) {
				r.printWarning.Do(func() {
					status := kubectl.GetPodStatus(pc.Pod)
					log.Warnf("Pod %s/%s has critical status: %s. DevSpace will continue waiting, but this operation might timeout", pc.Pod.Namespace, pc.Pod.Name, status)
				})
			}

			if !isWaitConditionTrue(hook.Wait, pc) {
				return false, nil
			}
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func isWaitConditionTrue(condition *latest.HookWaitConfig, podContainer *selector.SelectedPodContainer) bool {
	if selector.IsPodTerminating(podContainer.Pod) {
		return false
	}

	for _, cs := range podContainer.Pod.Status.InitContainerStatuses {
		if cs.Name == podContainer.Container.Name {
			if condition.Running && cs.State.Running != nil && cs.Ready {
				return true
			}
			if condition.TerminatedWithCode != nil && cs.State.Terminated != nil && cs.State.Terminated.ExitCode == *condition.TerminatedWithCode {
				return true
			}
		}
	}
	for _, cs := range podContainer.Pod.Status.ContainerStatuses {
		if cs.Name == podContainer.Container.Name {
			if condition.Running && cs.State.Running != nil && cs.Ready {
				return true
			}
			if condition.TerminatedWithCode != nil && cs.State.Terminated != nil && cs.State.Terminated.ExitCode == *condition.TerminatedWithCode {
				return true
			}
		}
	}

	return false
}
