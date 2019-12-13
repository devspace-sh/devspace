package cloud

import (
	"sync"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"

	"github.com/pkg/errors"
)

var cacheMutex sync.Mutex

// CacheSpace caches a given space and service account
func (p *provider) CacheSpace(space *latest.Space, serviceAccount *latest.ServiceAccount) error {
	now := time.Now()

	// Create cache object
	cachedSpace := &latest.SpaceCache{
		Space:          space,
		ServiceAccount: serviceAccount,

		KubeContext: GetKubeContextNameFromSpace(space.Name, space.ProviderName),
		Expires:     now.Add(time.Hour).Unix(),
	}

	if p.Spaces == nil {
		p.Spaces = map[int]*latest.SpaceCache{}
	}
	if p.Spaces[space.SpaceID] != nil {
		cachedSpace.LastResume = p.Spaces[space.SpaceID].LastResume
	}

	err := p.pruneCache()
	if err != nil {
		return errors.Wrap(err, "prune cache")
	}

	p.Spaces[space.SpaceID] = cachedSpace

	// Save the provider config
	return p.Save()
}

// pruneCache prunes the saved space information
func (p *provider) pruneCache() error {
	rawConfig, err := kubeconfig.LoadConfig().RawConfig()
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	for key, space := range p.Spaces {
		if _, ok := rawConfig.Contexts[space.KubeContext]; !ok {
			delete(p.Spaces, key)
		}
	}

	return nil
}

// GetAndUpdateSpaceCache retrieves space information from the providers.yaml and updates the space if necessary
func (p *provider) GetAndUpdateSpaceCache(spaceID int, forceUpdate bool) (*latest.SpaceCache, bool, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	now := time.Now()

	// Check if we have the space in the cache
	if forceUpdate == false && p.Spaces != nil && p.Spaces[spaceID] != nil {
		if now.Before(time.Unix(p.Spaces[spaceID].Expires, 0)) {
			return p.Spaces[spaceID], false, nil
		}
	}

	// Update space
	space, err := p.client.GetSpace(spaceID)
	if err != nil {
		return nil, false, errors.Wrap(err, "get space")
	}

	// Get cluster key
	key, err := p.GetClusterKey(space.Cluster)
	if err != nil {
		return nil, false, errors.Wrap(err, "get cluster key")
	}

	// Get service account token
	serviceAccount, err := p.client.GetServiceAccount(space, key)
	if err != nil {
		return nil, false, errors.Wrap(err, "get service account")
	}

	// Save cached space to config
	err = p.CacheSpace(space, serviceAccount)
	if err != nil {
		return nil, false, err
	}

	return p.Spaces[spaceID], true, nil
}
