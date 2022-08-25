package v2

import (
	"fmt"
	"time"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/helm/types"
)

// Client implements Interface
type Client struct {
	Releases []*types.Release
}

func (f *Client) DownloadChart(ctx devspacecontext.Context, helmConfig *latest.HelmConfig) (string, error) {
	return "", nil
}

// UpdateRepos implements interface
func (f *Client) UpdateRepos() error {
	return nil
}

// DeleteRelease deletes a helm release and optionally purges it
func (f *Client) DeleteRelease(ctx devspacecontext.Context, releaseName string, releaseNamespace string) error {
	for i, release := range f.Releases {
		if release.Name == releaseName {
			f.Releases = append(f.Releases[:i], f.Releases[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("release %s not found", releaseName)
}

// ListReleases lists all helm Releases
func (f *Client) ListReleases(ctx devspacecontext.Context, releaseNamespace string) ([]*types.Release, error) {
	return f.Releases, nil
}

// InstallChart implements interface
func (f *Client) InstallChart(ctx devspacecontext.Context, releaseName string, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
	for _, release := range f.Releases {
		if release.Name == releaseName {
			return release, nil
		}
	}

	newRelease := &types.Release{
		Name:         releaseName,
		Namespace:    releaseNamespace,
		Revision:     "1",
		Status:       "testStatus",
		LastDeployed: time.Now().String(),
	}

	f.Releases = append(f.Releases, newRelease)

	return newRelease, nil
}

// Template implements interface
func (f *Client) Template(ctx devspacecontext.Context, releaseName, releaseNamespace string, values map[string]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	return "", nil
}
