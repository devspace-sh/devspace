package patch

import (
	"context"
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	yaml2 "github.com/ghodss/yaml"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

const (
	RevertPatchAnnotation = "devspace.sh/revert-patch"

	MatchedContainer = "devspace.sh/container"
	PatchedLabel     = "devspace.sh/patched"
)

type Patcher interface {
	// Patch selects a pod container and applies a patch if no patched pod was found
	Patch(ctx context.Context, client kubectl.Client, patchConfig *latest.TerminalPatch, options targetselector.Options, log log.Logger) (*kubectl.SelectedPodContainer, error)

	// RevertPatches will try to revert all applied patches by DevSpace in the target namespace
	RevertPatches(ctx context.Context, client kubectl.Client, log log.Logger) error
}

func NewPatcher() Patcher {
	return &patcher{}
}

type patcher struct{}

func (p *patcher) RevertPatches(ctx context.Context, client kubectl.Client, log log.Logger) error {
	log.StartWait("Apply revert patches...")
	defer log.StopWait()

	// find all deployments, replicasets and statefulsets that were patched and reverse them
	patchedReplicaSets, err := client.KubeClient().AppsV1().ReplicaSets(client.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: PatchedLabel + "=true"})
	if err != nil {
		return errors.Wrap(err, "list replica sets")
	}

	// replica sets
	for _, rs := range patchedReplicaSets.Items {
		if rs.Annotations == nil || rs.Annotations[RevertPatchAnnotation] == "" {
			continue
		} else if metav1.GetControllerOf(&rs) != nil {
			continue
		}

		_, err = client.KubeClient().AppsV1().ReplicaSets(client.Namespace()).Patch(ctx, rs.Name, types.MergePatchType, []byte(rs.Annotations[RevertPatchAnnotation]), metav1.PatchOptions{})
		if err != nil {
			log.Warnf("Error patching replica set %s: %v", rs.Name, err)
		} else {
			log.Donef("Successfully reverted patched replica set %s", rs.Name)
		}
	}

	// find all deployments that were patched and reverse them
	patchedDeployments, err := client.KubeClient().AppsV1().Deployments(client.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: PatchedLabel + "=true"})
	if err != nil {
		return errors.Wrap(err, "list deployments")
	}

	// deployments
	for _, rs := range patchedDeployments.Items {
		if rs.Annotations == nil || rs.Annotations[RevertPatchAnnotation] == "" {
			continue
		}

		_, err = client.KubeClient().AppsV1().Deployments(client.Namespace()).Patch(ctx, rs.Name, types.MergePatchType, []byte(rs.Annotations[RevertPatchAnnotation]), metav1.PatchOptions{})
		if err != nil {
			log.Warnf("Error patching deployment %s: %v", rs.Name, err)
		} else {
			log.Donef("Successfully reverted patched deployment %s", rs.Name)
		}
	}

	// find all stateful sets that were patched and reverse them
	patchedStatefulSets, err := client.KubeClient().AppsV1().StatefulSets(client.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: PatchedLabel + "=true"})
	if err != nil {
		return errors.Wrap(err, "list stateful set")
	}

	// stateful sets
	for _, rs := range patchedStatefulSets.Items {
		if rs.Annotations == nil || rs.Annotations[RevertPatchAnnotation] == "" {
			continue
		}

		_, err = client.KubeClient().AppsV1().StatefulSets(client.Namespace()).Patch(ctx, rs.Name, types.MergePatchType, []byte(rs.Annotations[RevertPatchAnnotation]), metav1.PatchOptions{})
		if err != nil {
			log.Warnf("Error patching stateful set %s: %v", rs.Name, err)
		} else {
			log.Donef("Successfully reverted patched stateful set %s", rs.Name)
		}
	}

	if len(patchedReplicaSets.Items) == 0 && len(patchedDeployments.Items) == 0 && len(patchedStatefulSets.Items) == 0 {
		log.Info("Couldn't find any patched replica sets, deployments or stateful sets")
	}
	return nil
}

func (p *patcher) Patch(ctx context.Context, client kubectl.Client, patchConfig *latest.TerminalPatch, options targetselector.Options, log log.Logger) (*kubectl.SelectedPodContainer, error) {
	// check if there is a patched pod in the target namespace
	log.StartWait("Try to find patched pod...")
	defer log.StopWait()

	// try to find a single patched pod
	selectedPod, err := findSinglePatchedPod(ctx, client, options, time.Second*2, log)
	if err != nil {
		return nil, errors.Wrap(err, "find patched pod")
	} else if selectedPod != nil {
		return selectedPod, nil
	}

	// try to find a single patchable object
	log.StartWait("Try to find patchable pod...")
	target, container, err := findSinglePatchableObject(ctx, client, options, log)
	if err != nil {
		return nil, errors.Wrap(err, "find patchable object")
	}

	typeAccessor, err := meta.TypeAccessor(target)
	if err != nil {
		return nil, err
	}

	metaAccessor, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	// patch the object
	log.StartWait(fmt.Sprintf("Patching %s %s...", typeAccessor.GetKind(), metaAccessor.GetName()))
	err = patch(ctx, client, patchConfig, target, container)
	if err != nil {
		return nil, err
	}

	log.Donef("Successfully patched %s %s", typeAccessor.GetKind(), metaAccessor.GetName())

	// try to find a single patched pod again
	log.StartWait("Wait for patched pods to start...")
	selectedPod, err = findSinglePatchedPod(ctx, client, options, time.Minute, log)
	if err != nil {
		return nil, errors.Wrap(err, "find patched pod")
	} else if selectedPod != nil {
		return selectedPod, nil
	}

	return nil, fmt.Errorf("couldn't find a patched pod")
}

func patch(ctx context.Context, client kubectl.Client, patchConfig *latest.TerminalPatch, target runtime.Object, container string) error {
	var err error
	raw := convertToInterface(target)
	if len(patchConfig.Other) > 0 {
		raw, err = loader.ApplyPatchesOnObject(raw, patchConfig.Other)
		if err != nil {
			return errors.Wrap(err, "apply patches")
		}
	}

	// convert back
	rawJson := convertFromInterface(raw)

	// update object based on type
	switch target.(type) {
	case *appsv1.ReplicaSet:
		newReplicaSet := &appsv1.ReplicaSet{}
		err := json.Unmarshal(rawJson, newReplicaSet)
		if err != nil {
			return err
		}

		// modify template
		modifyPodTemplate(&newReplicaSet.Spec.Template, patchConfig.Image, container)

		// calculate revert patch
		err = setRevertPatch(target, newReplicaSet)
		if err != nil {
			return errors.Wrap(err, "set revert patch")
		}

		// update replica set
		_, err = client.KubeClient().AppsV1().ReplicaSets(newReplicaSet.Namespace).Update(ctx, newReplicaSet, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "update replica set")
		}

		return nil
	case *appsv1.Deployment:
		newDeployment := &appsv1.Deployment{}
		err := json.Unmarshal(rawJson, newDeployment)
		if err != nil {
			return err
		}

		// modify template
		modifyPodTemplate(&newDeployment.Spec.Template, patchConfig.Image, container)

		// calculate revert patch
		err = setRevertPatch(target, newDeployment)
		if err != nil {
			return errors.Wrap(err, "set revert patch")
		}

		// update deployment
		_, err = client.KubeClient().AppsV1().Deployments(newDeployment.Namespace).Update(ctx, newDeployment, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "update replica set")
		}

		return nil
	case *appsv1.StatefulSet:
		newStatefulSet := &appsv1.StatefulSet{}
		err := json.Unmarshal(rawJson, newStatefulSet)
		if err != nil {
			return err
		}

		// modify template
		modifyPodTemplate(&newStatefulSet.Spec.Template, patchConfig.Image, container)

		// calculate revert patch
		err = setRevertPatch(target, newStatefulSet)
		if err != nil {
			return errors.Wrap(err, "set revert patch")
		}

		// update statefulset
		_, err = client.KubeClient().AppsV1().StatefulSets(newStatefulSet.Namespace).Update(ctx, newStatefulSet, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "update replica set")
		}

		return nil
	}

	return nil
}

func setRevertPatch(oldObj runtime.Object, newObj metav1.Object) error {
	// set patched label
	labels := newObj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[PatchedLabel] = "true"
	newObj.SetLabels(labels)

	// create patch
	newObjBytes, err := json.Marshal(newObj)
	if err != nil {
		return err
	}
	oldObjBytes, err := json.Marshal(oldObj)
	if err != nil {
		return err
	}
	revertPatch, err := jsonpatch.CreateMergePatch(newObjBytes, oldObjBytes)
	if err != nil {
		return err
	}

	// set revert patch annotation
	annotations := newObj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[RevertPatchAnnotation] = string(revertPatch)
	newObj.SetAnnotations(annotations)
	return nil
}

func modifyPodTemplate(podTemplate *corev1.PodTemplateSpec, image string, container string) {
	// set annotations
	if podTemplate.ObjectMeta.Annotations == nil {
		podTemplate.ObjectMeta.Annotations = map[string]string{}
	}
	podTemplate.ObjectMeta.Annotations[MatchedContainer] = container

	// set labels
	if podTemplate.ObjectMeta.Labels == nil {
		podTemplate.ObjectMeta.Labels = map[string]string{}
	}
	podTemplate.ObjectMeta.Labels[PatchedLabel] = "true"

	// check if we should change image
	if image != "" {
		for i := range podTemplate.Spec.Containers {
			if podTemplate.Spec.Containers[i].Name == container {
				podTemplate.Spec.Containers[i].Image = image
			}
		}
	}
}

func convertFromInterface(inter map[interface{}]interface{}) []byte {
	out, err := yaml.Marshal(inter)
	if err != nil {
		panic(err)
	}

	retOut, err := yaml2.YAMLToJSON(out)
	if err != nil {
		panic(err)
	}

	return retOut
}

func convertToInterface(str runtime.Object) map[interface{}]interface{} {
	out, err := json.Marshal(str)
	if err != nil {
		panic(err)
	}

	ret := map[interface{}]interface{}{}
	err = yaml.Unmarshal(out, ret)
	if err != nil {
		panic(err)
	}

	return ret
}

func findSinglePatchableObject(ctx context.Context, client kubectl.Client, options targetselector.Options, log log.Logger) (runtime.Object, string, error) {
	targetSelector := targetselector.NewTargetSelector(client)
	options.SkipInitContainers = true
	options.FilterPod = nil
	options.FilterContainer = nil
	options.WaitingStrategy = targetselector.NewUntilNotTerminatingStrategy(time.Second * 2)
	options.Question = "Which pod do you want to open the terminal for?"

	// find pod / container
	container, err := targetSelector.SelectSingleContainer(ctx, options, log)
	if err != nil {
		return nil, "", err
	}

	// find patchable parent
	parent, err := getParent(ctx, client, container.Pod)
	if err != nil {
		return nil, "", err
	}

	return parent, container.Container.Name, nil
}

func findSinglePatchedPod(ctx context.Context, client kubectl.Client, options targetselector.Options, timeout time.Duration, log log.Logger) (*kubectl.SelectedPodContainer, error) {
	var selectedPod *kubectl.SelectedPodContainer
	err := wait.PollImmediate(time.Millisecond*500, timeout, func() (done bool, err error) {
		pods, err := kubectl.NewFilter(client).SelectPods(ctx, kubectl.Selector{
			LabelSelector: PatchedLabel + "=true",
			Namespace:     options.Namespace,
			// we filter out terminating pods
			FilterPod: func(p *corev1.Pod) bool {
				if p.DeletionTimestamp != nil || p.Annotations == nil || p.Annotations[MatchedContainer] == "" {
					return true
				}
				return false
			},
		})
		if err != nil {
			return false, err
		}

		// no pods found
		if len(pods) == 0 {
			return false, nil
		}

		// multiple pods
		if len(pods) > 1 {
			podNames := []string{}
			for _, p := range pods {
				podNames = append(podNames, p.Name)

			}

			answer, err := log.Question(&survey.QuestionOptions{
				Question: "Multiple patched pods found in namespace " + pods[0].Namespace,
				Options:  podNames,
				Sort:     true,
			})
			if err != nil {
				return false, err
			}
			for _, p := range pods {
				if p.Name != answer {

				}

				// search containers
				containerName := p.Annotations[MatchedContainer]
				for _, c := range p.Spec.Containers {
					if c.Name == containerName {
						selectedPod = &kubectl.SelectedPodContainer{
							Pod:       p,
							Container: &c,
						}
						return true, nil
					}
				}

				return false, fmt.Errorf("couldn't find container %s in pod %s", containerName, p.Name)
			}

			return false, fmt.Errorf("couldn't find pod %s", answer)
		}

		// search containers
		p := pods[0]
		containerName := p.Annotations[MatchedContainer]
		for _, c := range p.Spec.Containers {
			if c.Name == containerName {
				selectedPod = &kubectl.SelectedPodContainer{
					Pod:       p,
					Container: &c,
				}
				return true, nil
			}
		}

		return false, fmt.Errorf("couldn't find container %s in pod %s", containerName, p.Name)
	})
	if err == wait.ErrWaitTimeout {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return selectedPod, nil
}

func getParent(ctx context.Context, client kubectl.Client, pod *corev1.Pod) (runtime.Object, error) {
	controller := metav1.GetControllerOf(pod)
	if controller == nil {
		return nil, fmt.Errorf("pod was not created by a ReplicaSet, Deployment or StatefulSet, patching only works if pod was created by one of those resources")
	}

	// replica set / deployment ?
	if controller.Kind == "ReplicaSet" {
		// try to find the replica set, we ignore the group version for now
		replicaSet, err := client.KubeClient().AppsV1().ReplicaSets(pod.Namespace).Get(ctx, controller.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("unrecognized owning ReplicaSet %s group version: %s", controller.Name, controller.APIVersion)
			}

			return nil, err
		}

		replicaSetOwner := metav1.GetControllerOf(replicaSet)
		if replicaSetOwner == nil {
			return replicaSet, nil
		}

		// is deployment?
		if replicaSetOwner.Kind == "Deployment" {
			deployment, err := client.KubeClient().AppsV1().Deployments(pod.Namespace).Get(ctx, replicaSetOwner.Name, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return nil, fmt.Errorf("unrecognized owning Deployment %s group version: %s", replicaSetOwner.Name, replicaSetOwner.APIVersion)
				}

				return nil, err
			}

			// we stop here, if the Deployment is owned by something else we just ignore it for now
			return deployment, nil
		}

		return nil, fmt.Errorf("unrecognized owner of ReplicaSet %s: %s %s %s", replicaSet.Name, replicaSetOwner.Kind, replicaSetOwner.APIVersion, replicaSetOwner.Name)
	}

	// statefulset?
	if controller.Kind == "StatefulSet" {
		statefulSet, err := client.KubeClient().AppsV1().StatefulSets(pod.Namespace).Get(ctx, controller.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, fmt.Errorf("unrecognized owning StatefulSet %s group version: %s", controller.Name, controller.APIVersion)
			}

			return nil, err
		}

		return statefulSet, nil
	}

	return nil, fmt.Errorf("unrecognized owner of Pod %s: %s %s %s", pod.Name, controller.Kind, controller.APIVersion, controller.Name)
}
