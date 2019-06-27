package cloud

import (
	"fmt"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/util/log"

	"github.com/pkg/errors"
)

// ActivateSpace ...
func ActivateSpace(generatedConfig *generated.Config, loop bool, log log.Logger) error {
	if generatedConfig.CloudSpace != nil {
		p, err := GetProvider(&generatedConfig.CloudSpace.ProviderName, log)
		if err != nil {
			return err
		}

		space, err := p.GetSpace(generatedConfig.CloudSpace.SpaceID)
		if err != nil {
			return fmt.Errorf("Error retrieving Spaces details: %v", err)
		}

		err = p.ActivateSpace(space.SpaceID, space.Cluster)
		if err != nil {
			return errors.Wrap(err, "active space")
		}

		if loop {
			go func() {
				for {
					time.Sleep(time.Minute * 5)
					p.ActivateSpace(space.SpaceID, space.Cluster)
				}
			}()
		}
	}

	return nil
}

// ActivateSpace creates a user cluster with the given name
func (p *Provider) ActivateSpace(spaceID int, cluster *Cluster) error {
	key, err := p.GetClusterKey(cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster key")
	}

	// Do the request
	err = p.GrapqhlRequest(`
		mutation ($key:String, $spaceID: Int!){
			manager_activateSpace(key: $key, spaceID: $spaceID)
		}
	`, map[string]interface{}{
		"key":     key,
		"spaceID": spaceID,
	}, &struct {
		ActivateSpace bool `json:"manager_activateSpace"`
	}{})
	if err != nil {
		return err
	}

	return nil
}
