package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
)

// Client is the client interface for helm
type Client interface {
	InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*Release, error)
	Template(releaseName, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (string, error)
	DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig) error
	ListReleases(helmConfig *latest.HelmConfig) ([]*Release, error)
}

// Release is the helm release struct
type Release struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Revision     string `json:"revision"`
	LastDeployed string `json:"updated"`
}
