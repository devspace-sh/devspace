package targetselector

import (
	"strings"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
)

// SelectorParameter holds the information from the config and the command overrides
type SelectorParameter struct {
	ConfigParameter ConfigParameter
	CmdParameter    CmdParameter
}

// CmdParameter holds the parameter we receive from the command
type CmdParameter struct {
	LabelSelector string
	Namespace     string
	ContainerName string
	PodName       string
	Pick          *bool
	Interactive   bool
}

// ConfigParameter holds the parameter we receive from the config
type ConfigParameter struct {
	LabelSelector map[string]string
	Namespace     string
	ContainerName string
}

// GetNamespace retrieves the target namespace
func (t *SelectorParameter) GetNamespace(config *latest.Config, kubeClient kubectl.Client) (string, error) {
	if t.CmdParameter.Namespace != "" {
		return t.CmdParameter.Namespace, nil
	}
	if t.ConfigParameter.Namespace != "" {
		return t.ConfigParameter.Namespace, nil
	}

	return kubeClient.Namespace(), nil
}

// GetLabelSelector retrieves the label selector of the target
func (t *SelectorParameter) GetLabelSelector(config *latest.Config) (string, error) {
	if t.CmdParameter.LabelSelector != "" {
		return t.CmdParameter.LabelSelector, nil
	}
	if t.ConfigParameter.LabelSelector != nil {
		labelSelector := labelSelectorMapToString(t.ConfigParameter.LabelSelector)
		return labelSelector, nil
	}

	return "", nil
}

func labelSelectorMapToString(m map[string]string) string {
	labels := []string{}
	for key, value := range m {
		labels = append(labels, key+"="+value)
	}

	return strings.Join(labels, ",")
}

// GetPodName retrieves the pod name from the parameters
func (t *SelectorParameter) GetPodName() string {
	if t.CmdParameter.PodName != "" {
		return t.CmdParameter.PodName
	}

	return ""
}

// GetContainerName retrieves the container name from the parameters
func (t *SelectorParameter) GetContainerName() string {
	if t.CmdParameter.ContainerName != "" {
		return t.CmdParameter.ContainerName
	}
	if t.ConfigParameter.ContainerName != "" {
		return t.ConfigParameter.ContainerName
	}

	return ""
}
