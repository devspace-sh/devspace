package services

import (
	"context"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/loader/variable/runtime"
	"github.com/loft-sh/devspace/pkg/devspace/imageselector"
	"strings"
	"sync"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const k8sComponentLabel = "app.kubernetes.io/component"

type LogManager interface {
	Start() error
}

type logManager struct {
	client kubectl.Client

	imageSelectors []namespacedImageSelector
	labelSelectors []latest.LogsSelector

	tail int64

	interrupt chan error
	output    log.Logger

	activeLogsMutex sync.Mutex
	activeLogs      map[string]activeLog
}

type namespacedImageSelector struct {
	imageselector.ImageSelector

	Namespace string
}

func NewLogManager(client kubectl.Client, config config.Config, dependencies []types.Dependency, interrupt chan error, out log.Logger) (LogManager, error) {
	if config == nil || config.Config() == nil || config.Generated() == nil {
		return nil, fmt.Errorf("no devspace config loaded")
	}

	// get config
	var (
		c              = config.Config()
		tail           *int64
		imageSelectors = []namespacedImageSelector{}
		labelSelectors = []latest.LogsSelector{}
	)

	if c.Dev.Logs != nil {
		if c.Dev.Logs.ShowLast != nil {
			tail = ptr.Int64(int64(*c.Dev.Logs.ShowLast))
		}

		// resolve selectors
		for _, selector := range c.Dev.Logs.Selectors {
			if selector.ImageSelector != "" {
				imageSelector, err := runtime.NewRuntimeResolver(true).FillRuntimeVariablesAsImageSelector(selector.ImageSelector, config, dependencies)
				if err != nil {
					return nil, err
				}

				imageSelectors = append(imageSelectors, namespacedImageSelector{
					ImageSelector: *imageSelector,
					Namespace:     selector.Namespace,
				})
			} else {
				labelSelectors = append(labelSelectors, selector)
			}
		}
	}

	// if we don't have any selectors, use the current images as selector
	if len(labelSelectors)+len(imageSelectors) == 0 {
		for configImageName := range c.Images {
			selector, err := imageselector.Resolve(configImageName, config, dependencies)
			if err != nil {
				return nil, err
			} else if selector != nil {
				imageSelectors = append(imageSelectors, namespacedImageSelector{
					ImageSelector: *selector,
				})
			}
		}
	}

	if tail == nil {
		tail = ptr.Int64(50)
	}
	return &logManager{
		client:         client,
		imageSelectors: imageSelectors,
		labelSelectors: labelSelectors,
		interrupt:      interrupt,
		output:         out,
		tail:           *tail,
		activeLogs:     map[string]activeLog{},
	}, nil
}

type activeLog struct {
	cancelCtx func()
	log       log.Logger
}

func (l *logManager) Start() error {
	l.output.Info("Starting log streaming")
	for {
		targets, err := l.gatherPods()
		if err != nil {
			l.output.Errorf("Error gathering target pods for log streaming: %v", err)
			time.Sleep(time.Second * 15)
			continue
		}

		l.activeLogsMutex.Lock()
		for _, t := range targets {
			_, ok := l.activeLogs[t.key]
			if ok {
				continue
			}

			splitted := strings.Split(t.key, "/")
			namespace, pod, container := splitted[0], splitted[1], splitted[2]

			logsContext, cancel := context.WithCancel(context.Background())
			logsLog := log.NewPrefixLogger("["+t.name+"] ", log.Colors[(len(log.Colors)-1)-(len(l.activeLogs)%len(log.Colors))], l.output)
			go func() {
				logsLog.Infof("Start streaming logs for %s", t.key)
				reader, err := l.client.Logs(logsContext, namespace, pod, container, false, &l.tail, true)
				if err != nil {
					logsLog.Warnf("Error streaming logs: %v", err)
				}

				if reader != nil {
					scanner := scanner.NewScanner(reader)
					for scanner.Scan() {
						logsLog.Info(scanner.Text())
					}
					if scanner.Err() != nil && scanner.Err() != context.Canceled {
						logsLog.Warnf("Error streaming logs for %s: %v", t.key, scanner.Err())
					} else {
						logsLog.Infof("End streaming logs for %s", t.key)
					}
				}

				l.activeLogsMutex.Lock()
				delete(l.activeLogs, t.key)
				l.activeLogsMutex.Unlock()
			}()

			l.activeLogs[t.key] = activeLog{
				cancelCtx: cancel,
				log:       logsLog,
			}
		}
		l.activeLogsMutex.Unlock()

		select {
		case err := <-l.interrupt:
			// cleanup
			l.activeLogsMutex.Lock()
			for _, v := range l.activeLogs {
				v.cancelCtx()
			}
			l.activeLogsMutex.Unlock()
			return err
		case <-time.After(time.Second * 5):
		}
	}
}

type podInfo struct {
	key  string
	name string
}

func (l *logManager) gatherPods() ([]podInfo, error) {
	returnList := []podInfo{}
	selectors := []selector.Selector{}
	filterPod := func(p *k8sv1.Pod) bool {
		return kubectl.GetPodStatus(p) != "Running"
	}

	// first gather all pods by image
	for _, s := range l.imageSelectors {
		selectors = append(selectors, selector.Selector{
			ImageSelector:      []imageselector.ImageSelector{s.ImageSelector},
			Namespace:          s.Namespace,
			FilterPod:          filterPod,
			SkipInitContainers: true,
		})
	}

	// now gather all pods by label selector
	for _, s := range l.labelSelectors {
		labelSelector := ""
		if s.LabelSelector != nil {
			labelSelector = labels.Set(s.LabelSelector).String()
		}

		selectors = append(selectors, selector.Selector{
			LabelSelector:      labelSelector,
			ContainerName:      s.ContainerName,
			Namespace:          s.Namespace,
			FilterPod:          filterPod,
			SkipInitContainers: true,
		})
	}

	if len(selectors) > 0 {
		selectedPodsContainers, err := selector.NewFilter(l.client).SelectContainers(context.TODO(), selectors...)
		if err != nil {
			return nil, err
		}

		for _, podContainer := range selectedPodsContainers {
			returnList = append(returnList, podInfo{
				key:  key(podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name),
				name: getDisplayName(l.client, podContainer),
			})
		}
	}

	return returnList, nil
}

func getDisplayName(client kubectl.Client, podContainer *selector.SelectedPodContainer) string {
	controller := metav1.GetControllerOf(podContainer.Pod)

	// pod name by default, or deployment or statefulset name if found
	name := podContainer.Pod.Name
	if componentLabel, ok := podContainer.Pod.Labels[k8sComponentLabel]; ok {
		name = componentLabel
	} else if controller != nil && controller.Kind == "ReplicaSet" && controller.APIVersion == appsv1.SchemeGroupVersion.String() {
		name = controller.Name

		rs, err := client.KubeClient().AppsV1().ReplicaSets(podContainer.Pod.Namespace).Get(context.TODO(), controller.Name, metav1.GetOptions{})
		if err == nil {
			controller = metav1.GetControllerOf(rs)
			if controller != nil && controller.Kind == "Deployment" && controller.APIVersion == appsv1.SchemeGroupVersion.String() {
				name = controller.Name
			}
		}
	} else if controller != nil && controller.Kind == "StatefulSet" && controller.APIVersion == appsv1.SchemeGroupVersion.String() {
		name = controller.Name
	}

	// if the pod has multiple containers, we mark the container
	if len(podContainer.Pod.Spec.Containers) > 1 {
		name += ":" + podContainer.Container.Name
	}

	// if the pod is in another namespace we add the namespace
	if podContainer.Pod.Namespace != client.Namespace() {
		name = podContainer.Pod.Namespace + ":" + name
	}

	return name
}

func key(namespace string, pod string, container string) string {
	return namespace + "/" + pod + "/" + container
}
