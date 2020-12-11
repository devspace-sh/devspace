package v2

import (
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/helm/types"
	"time"
)

// Client implements Interface
type Client struct {
	Releases []*types.Release
}

// UpdateRepos implements interface
func (f *Client) UpdateRepos() error {
	return nil
}

// DeleteRelease deletes a helm release and optionally purges it
func (f *Client) DeleteRelease(releaseName string, releaseNamespace string, helmConfig *latest.HelmConfig) error {
	for i, release := range f.Releases {
		if release.Name == releaseName {
			f.Releases = append(f.Releases[:i], f.Releases[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("Release %s not found", releaseName)
}

// ListReleases lists all helm Releases
func (f *Client) ListReleases(helmConfig *latest.HelmConfig) ([]*types.Release, error) {
	return f.Releases, nil
}

// InstallChart implements interface
func (f *Client) InstallChart(releaseName string, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (*types.Release, error) {
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
func (f *Client) Template(releaseName, releaseNamespace string, values map[interface{}]interface{}, helmConfig *latest.HelmConfig) (string, error) {
	return "", nil
}
