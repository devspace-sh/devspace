package services

import (
	"bufio"
	"context"
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
	"sync"
	"time"
)

const k8sComponentLabel = "app.kubernetes.io/component"

type LogManager interface {
	Start() error
}

type logManager struct {
	client kubectl.Client

	imageNameSelectors []string
	labelSelectors     []latest.LogsSelector

	tail int64

	interrupt chan error
	output    log.Logger

	activeLogsMutex sync.Mutex
	activeLogs      map[string]activeLog
}

func NewLogManager(client kubectl.Client, config *latest.Config, generatedConfig *generated.Config, interrupt chan error, out log.Logger) (LogManager, error) {
	if config == nil || generatedConfig == nil {
		return nil, fmt.Errorf("no devspace config loaded")
	}

	// Build an image selector
	imageSelector := []string{}
	if config.Dev != nil && config.Dev.Logs != nil {
		for _, configImageName := range config.Dev.Logs.Images {
			imageSelector = append(imageSelector, targetselector.ImageSelectorFromConfig(configImageName, config, generatedConfig.GetActive())...)
		}
	} else {
		for configImageName := range config.Images {
			imageSelector = append(imageSelector, targetselector.ImageSelectorFromConfig(configImageName, config, generatedConfig.GetActive())...)
		}
	}

	// Show last log lines
	var tail *int64
	if config.Dev != nil && config.Dev.Logs != nil && config.Dev.Logs.ShowLast != nil {
		tail = ptr.Int64(int64(*config.Dev.Logs.ShowLast))
	}

	var selectors []latest.LogsSelector
	if config.Dev != nil && config.Dev.Logs != nil {
		selectors = config.Dev.Logs.Selectors
	}
	if tail == nil {
		tail = ptr.Int64(50)
	}

	return &logManager{
		client:             client,
		imageNameSelectors: imageSelector,
		labelSelectors:     selectors,
		interrupt:          interrupt,
		output:             out,
		tail:               *tail,
		activeLogs:         map[string]activeLog{},
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
			l.output.Errorf("Error gathering target pods: %v", err)
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
			logsLog := log.NewDefaultPrefixLogger("["+t.name+"] ", l.output)
			go func() {
				logsLog.Infof("Start streaming logs for %s", t.key)
				reader, err := l.client.Logs(logsContext, namespace, pod, container, false, &l.tail, true)
				if err != nil {
					logsLog.Warnf("Error streaming logs: %v", err)
				}

				if reader != nil {
					scanner := bufio.NewScanner(reader)
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

	return nil
}

type podInfo struct {
	key  string
	name string
}

func (l *logManager) gatherPods() ([]podInfo, error) {
	returnList := []podInfo{}
	selectors := []kubectl.Selector{}
	filterPod := func(p *k8sv1.Pod) bool {
		return kubectl.GetPodStatus(p) != "Running"
	}

	// first gather all pods by image
	if len(l.imageNameSelectors) > 0 {
		selectors = append(selectors, kubectl.Selector{
			ImageSelector:      l.imageNameSelectors,
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

		selectors = append(selectors, kubectl.Selector{
			LabelSelector:      labelSelector,
			ContainerName:      s.ContainerName,
			Namespace:          s.Namespace,
			FilterPod:          filterPod,
			SkipInitContainers: true,
		})
	}

	if len(selectors) > 0 {
		selectedPodsContainers, err := kubectl.NewFilter(l.client).SelectContainers(context.TODO(), selectors...)
		if err != nil {
			return nil, err
		}

		for _, podContainer := range selectedPodsContainers {
			prefix := podContainer.Pod.Name
			if componentLabel, ok := podContainer.Pod.Labels[k8sComponentLabel]; ok {
				prefix = componentLabel
			}
			if len(podContainer.Pod.Spec.Containers) > 1 {
				prefix += ":" + podContainer.Container.Name
			}
			if podContainer.Pod.Namespace != l.client.Namespace() {
				prefix = podContainer.Pod.Namespace + ":" + prefix
			}

			returnList = append(returnList, podInfo{
				key:  key(podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name),
				name: prefix,
			})
		}
	}

	return returnList, nil
}

func key(namespace string, pod string, container string) string {
	return namespace + "/" + pod + "/" + container
}
