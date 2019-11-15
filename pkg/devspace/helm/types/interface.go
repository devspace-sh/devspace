package types

import (
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

// Client is the client interface for helm
type Client interface {
	InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*Release, error)
	DeleteRelease(releaseName string, purge bool) error
	ListReleases() ([]*Release, error)
}

// Release is the helm release struct
type Release struct {
	Name         string
	Namespace    string
	Status       string
	Version      int32
	LastDeployed time.Time
}
