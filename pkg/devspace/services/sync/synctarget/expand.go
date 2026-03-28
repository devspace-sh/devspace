package synctarget

import (
	"context"
	"fmt"
	"sort"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	kubeselector "github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Target struct {
	Selector  targetselector.TargetSelector
	Namespace string
	Pod       string
	Container string
}

// ReplicasEnabled returns true if syncReplicas is enabled on the config.
func ReplicasEnabled(syncConfig *latest.SyncConfig) bool {
	return syncConfig != nil && syncConfig.SyncReplicas
}

// ConfigForIndex returns sync config for replica index i (0 = primary, others upload-only).
func ConfigForIndex(syncConfig *latest.SyncConfig, index int) *latest.SyncConfig {
	copied := *syncConfig
	if index > 0 {
		copied.DisableDownload = true
		copied.OnUpload = nil
	}
	return &copied
}

// BuildTargets returns sync targets; with syncReplicas, one per pod in the deployment.
func BuildTargets(ctx context.Context, lg log.Logger, client kubectl.Client, selector targetselector.TargetSelector, syncConfig *latest.SyncConfig) ([]Target, error) {
	targets := []Target{{Selector: selector}}
	if !ReplicasEnabled(syncConfig) {
		return targets, nil
	}

	primary, err := selector.SelectSingleContainer(ctx, client, lg)
	if err != nil {
		return nil, err
	}
	if primary == nil {
		return nil, fmt.Errorf("couldn't find a pod / container with the configured selector")
	}

	deploymentName, err := DeploymentName(ctx, client, primary.Pod)
	if err != nil {
		return nil, err
	}
	if deploymentName == "" {
		return nil, fmt.Errorf("pod %s/%s is not part of a deployment", primary.Pod.Namespace, primary.Pod.Name)
	}

	deployment, err := client.KubeClient().AppsV1().Deployments(primary.Pod.Namespace).Get(ctx, deploymentName, metav1.GetOptions{})
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

	podList, err := client.KubeClient().CoreV1().Pods(primary.Pod.Namespace).List(ctx, metav1.ListOptions{
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
		podDeploymentName, err := DeploymentName(ctx, client, &pod)
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

	targets = make([]Target, 0, len(pods))
	for _, container := range pods {
		targets = append(targets, Target{
			Selector: targetselector.NewTargetSelector(
				targetselector.NewOptionsFromFlags(container.Container.Name, "", nil, container.Pod.Namespace, container.Pod.Name),
			),
			Namespace: container.Pod.Namespace,
			Pod:       container.Pod.Name,
			Container: container.Container.Name,
		})
	}

	lg.Infof("syncReplicas enabled: starting %d sync targets for path %s", len(targets), syncConfig.Path)
	return targets, nil
}

// DeploymentName returns the owning Deployment name for a pod, if any.
func DeploymentName(ctx context.Context, client kubectl.Client, pod *corev1.Pod) (string, error) {
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
