package types

import (
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
)

// Client is the client interface for helm
type Client interface {
	DownloadChart(ctx devspacecontext.Context, helmConfig *latest.HelmConfig) (string, error)
	InstallChart(ctx devspacecontext.Context, releaseName string, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (*Release, error)
	Template(ctx devspacecontext.Context, releaseName, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (string, error)
	DeleteRelease(ctx devspacecontext.Context, releaseName string, releaseNamespace string) error
	ListReleases(ctx devspacecontext.Context, releaseNamespace string) ([]*Release, error)
}

// Release is the helm release struct
type Release struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Revision     string `json:"revision"`
	LastDeployed string `json:"updated"`
}
