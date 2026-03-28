package sync

import (
	"context"
	"fmt"
	"sort"
	stdsync "sync"

	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	kubeselector "github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/tomb"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type syncTargetSelector struct {
	selector  targetselector.TargetSelector
	namespace string
	pod       string
	container string
}

// StartSyncFromCmd starts a new sync from command
func StartSyncFromCmd(ctx devspacecontext.Context, selector targetselector.TargetSelector, name string, syncConfig *latest.SyncConfig, noWatch bool) error {
	ctx, parent := ctx.WithNewTomb()
	<-parent.NotifyGo(func() error {
		parent.Go(func() error {
			<-ctx.Context().Done()
			return nil
		})
		return startSync(ctx, name, "", syncConfig, selector, nil, parent)
	})

	// Handle no watch
	if noWatch {
		select {
		case <-parent.Dead():
			return parent.Err()
		default:
			parent.Kill(nil)
			_ = parent.Wait()
			return nil
		}
	}

	// Handle interrupt
	select {
	case <-parent.Dead():
		return parent.Err()
	case <-ctx.Context().Done():
		_ = parent.Wait()
		return nil
	}
}

func buildTargetSelectors(ctx devspacecontext.Context, selector targetselector.TargetSelector, syncConfig *latest.SyncConfig) ([]syncTargetSelector, error) {
	targets := []syncTargetSelector{{
		selector: selector,
	}}
	if !syncReplicasEnabled(syncConfig) {
		return targets, nil
	}

	primary, err := selector.SelectSingleContainer(ctx.Context(), ctx.KubeClient(), ctx.Log())
	if err != nil {
		return nil, err
	}
	if primary == nil {
		return nil, fmt.Errorf("couldn't find a pod / container with the configured selector")
	}

	deploymentName, err := findDeploymentName(ctx.Context(), ctx.KubeClient(), primary.Pod)
	if err != nil {
		return nil, err
	}
	if deploymentName == "" {
		return nil, fmt.Errorf("pod %s/%s is not part of a deployment", primary.Pod.Namespace, primary.Pod.Name)
	}

	deployment, err := ctx.KubeClient().KubeClient().AppsV1().Deployments(primary.Pod.Namespace).Get(ctx.Context(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if deployment.Spec.Selector == nil {
		return nil, fmt.Errorf("deployment %s/%s has no selector", deployment.Namespace, deployment.Name)
	}
	deploymentSelector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}

	podList, err := ctx.KubeClient().KubeClient().CoreV1().Pods(primary.Pod.Namespace).List(ctx.Context(), metav1.ListOptions{
		LabelSelector: deploymentSelector.String(),
	})
	if err != nil {
		return nil, err
	}
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found in deployment %s/%s", primary.Pod.Namespace, deployment.Name)
	}

	pods := make([]*kubeselector.SelectedPodContainer, 0, len(podList.Items))
	for i := range podList.Items {
		pod := podList.Items[i]
		podDeploymentName, err := findDeploymentName(ctx.Context(), ctx.KubeClient(), &pod)
		if err != nil {
			return nil, err
		}
		if podDeploymentName != deploymentName {
			continue
		}
		pods = append(pods, &kubeselector.SelectedPodContainer{
			Pod:       &pod,
			Container: primary.Container,
		})
	}
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found in deployment %s/%s", primary.Pod.Namespace, deployment.Name)
	}

	sort.Slice(pods, func(i, j int) bool {
		return kubeselector.SortContainersByNewest(pods, i, j)
	})

	targets = make([]syncTargetSelector, 0, len(pods))
	for _, container := range pods {
		targets = append(targets, syncTargetSelector{
			selector: targetselector.NewTargetSelector(
				targetselector.NewOptionsFromFlags(container.Container.Name, "", nil, container.Pod.Namespace, container.Pod.Name),
			),
			namespace: container.Pod.Namespace,
			pod:       container.Pod.Name,
			container: container.Container.Name,
		})
	}

	ctx.Log().Infof("syncReplicas enabled: starting %d sync targets for path %s", len(targets), syncConfig.Path)
	return targets, nil
}

func findDeploymentName(ctx context.Context, client kubectl.Client, pod *corev1.Pod) (string, error) {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "Deployment" {
			return ownerRef.Name, nil
		}
		if ownerRef.Kind == "ReplicaSet" {
			replicaSet, err := client.KubeClient().AppsV1().ReplicaSets(pod.Namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				return "", err
			}

			for _, rsOwnerRef := range replicaSet.OwnerReferences {
				if rsOwnerRef.Kind == "Deployment" {
					return rsOwnerRef.Name, nil
				}
			}
		}
	}

	return "", nil
}

func configForTarget(syncConfig *latest.SyncConfig, targetIndex int) *latest.SyncConfig {
	copied := *syncConfig
	if targetIndex > 0 {
		copied.DisableDownload = true
		copied.OnUpload = nil
	}

	return &copied
}

func syncReplicasEnabled(syncConfig *latest.SyncConfig) bool {
	return syncConfig.SyncReplicas
}

// StartSync starts the syncing functionality
func StartSync(ctx devspacecontext.Context, devPod *latest.DevPod, selector targetselector.TargetSelector, parent *tomb.Tomb) (retErr error) {
	if ctx == nil || ctx.Config() == nil || ctx.Config().Config() == nil {
		return fmt.Errorf("DevSpace config is nil")
	}

	// init done array is used to track when sync was initialized
	initDoneArray := []chan struct{}{}
	loader.EachDevContainer(devPod, func(devContainer *latest.DevContainer) bool {
		starter := sync.NewDelayedContainerStarter()

		// make sure we add all the sync paths that need to wait for initial start
		for _, syncConfig := range devContainer.Sync {
			if syncConfig.SyncReplicas {
				continue
			}
			if syncConfig.StartContainer || (syncConfig.OnUpload != nil && syncConfig.OnUpload.RestartContainer) {
				starter.Inc()
			}
		}

		// now start the sync paths
		for _, syncConfig := range devContainer.Sync {
			// start a new go routine in the tomb
			s := syncConfig
			syncCtx := ctx
			var cancel context.CancelFunc
			if s.NoWatch {
				var cancelCtx context.Context
				cancelCtx, cancel = context.WithCancel(syncCtx.Context())
				syncCtx = syncCtx.WithContext(cancelCtx)
			}
			initDone := parent.NotifyGo(func() error {
				if cancel != nil {
					defer cancel()
				}

				return startSync(syncCtx, devPod.Name, string(devContainer.Arch), s, selector.WithContainer(devContainer.Container), starter, parent)
			})
			initDoneArray = append(initDoneArray, initDone)

			// every five we wait
			if len(initDoneArray)%5 == 0 {
				for _, initDone := range initDoneArray {
					<-initDone
				}
			}
		}
		return true
	})

	// wait for init chans to be finished
	for _, initDone := range initDoneArray {
		<-initDone
	}
	return nil
}

func startSync(ctx devspacecontext.Context, name, arch string, syncConfig *latest.SyncConfig, selector targetselector.TargetSelector, starter sync.DelayedContainerStarter, parent *tomb.Tomb) error {
	targetSelectors, err := buildTargetSelectors(ctx, selector, syncConfig)
	if err != nil {
		return err
	}

	// Multiple replica targets must not share the dev session tomb: the sync controller
	// calls parent.Kill on normal completion and errors, which would tear down every
	// other sync (including other dev pods) mid-flight. Use an isolated tomb per target.
	if len(targetSelectors) == 1 {
		return startSyncOneTarget(ctx, name, arch, syncConfig, starter, parent, targetSelectors, 0)
	}

	var wg stdsync.WaitGroup
	errs := make([]error, len(targetSelectors))
	for i := range targetSelectors {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			var subTomb tomb.Tomb
			errs[i] = startSyncOneTarget(ctx, name, arch, syncConfig, starter, &subTomb, targetSelectors, i)
		}()
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func startSyncOneTarget(
	ctx devspacecontext.Context,
	name, arch string,
	syncConfig *latest.SyncConfig,
	starter sync.DelayedContainerStarter,
	syncParent *tomb.Tomb,
	targetSelectors []syncTargetSelector,
	i int,
) error {
	target := targetSelectors[i]
	syncConfigForTarget := configForTarget(syncConfig, i)
	if syncReplicasEnabled(syncConfig) && target.pod != "" && target.container != "" {
		ctx.Log().Infof(
			"Sync target %d/%d: %s/%s:%s (disableUpload=%t disableDownload=%t)",
			i+1,
			len(targetSelectors),
			target.namespace,
			target.pod,
			target.container,
			syncConfigForTarget.DisableUpload,
			syncConfigForTarget.DisableDownload,
		)
	}

	effectiveStarter := starter
	if syncReplicasEnabled(syncConfig) && (syncConfig.StartContainer || (syncConfig.OnUpload != nil && syncConfig.OnUpload.RestartContainer)) {
		ts := sync.NewDelayedContainerStarter()
		ts.Inc()
		effectiveStarter = ts
	}

	options := &Options{
		Name:       name,
		Selector:   target.selector,
		SyncConfig: syncConfigForTarget,
		Arch:       arch,
		Starter:    effectiveStarter,

		RestartOnError: true,
		Verbose:        ctx.Log().GetLevel() == logrus.DebugLevel,
	}

	if syncConfigForTarget.PrintLogs || ctx.Log().GetLevel() == logrus.DebugLevel {
		options.SyncLog = ctx.Log()
	} else {
		options.SyncLog = logpkg.GetDevPodFileLogger(name)
	}

	return NewController().Start(ctx, options, syncParent)
}
