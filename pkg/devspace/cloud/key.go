package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	"github.com/pkg/errors"
)

// GetClusterKey makes sure there is a correct key for the given cluster id
func (p *Provider) GetClusterKey(cluster *Cluster) (string, error) {
	if cluster.Owner == nil {
		return "", nil
	}

	key, ok := p.ClusterKey[cluster.ClusterID]
	if ok == false {
		if len(p.ClusterKey) > 0 {
			for _, clusterKey := range p.ClusterKey {
				key = clusterKey
				break
			}

		} else {
			return p.AskForEncryptionKey(cluster)
		}
	}

	// Verifies the cluster key
	err := p.VerifyKey(key, cluster.ClusterID)
	if err != nil {
		return p.AskForEncryptionKey(cluster)
	}

	// Save the key if it was not there
	if _, ok := p.ClusterKey[cluster.ClusterID]; ok == false {
		p.ClusterKey[cluster.ClusterID] = key

		// Save provider config
		err := p.Save()
		if err != nil {
			return "", errors.Wrap(err, "save provider")
		}

		// Save config
		return key, nil
	}

	return key, nil
}

// AskForEncryptionKey asks the user for his her encryption key and verifies that the key is correct
func (p *Provider) AskForEncryptionKey(cluster *Cluster) (string, error) {
	// Wait till user enters the correct key
	for true {
		key := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
			Question:               "Please enter your encryption key for cluster " + cluster.Name,
			ValidationRegexPattern: "^.{6,32}$",
			IsPassword:             true,
		})

		hashedKey, err := hash.BcryptPassword(key)
		if err != nil {
			return "", errors.Wrap(err, "bcrypt key")
		}

		err = p.VerifyKey(hashedKey, cluster.ClusterID)
		if err != nil {
			log.Errorf("Encryption key is incorrect. Please try again")
			continue
		}

		p.ClusterKey[cluster.ClusterID] = hashedKey

		// Save config
		err = p.Save()
		if err != nil {
			return "", errors.Wrap(err, "save provider")
		}

		return hashedKey, nil
	}

	// We should never reach that
	return "", nil
}

// VerifyKey verifies the given key for the given cluster
func (p *Provider) VerifyKey(key string, clusterID int) error {
	clusterUser, err := p.GetClusterUser(clusterID)
	if err != nil {
		return errors.Wrap(err, "get cluster user")
	}

	// Response struct
	response := struct {
		VerifyKey bool `json:"manager_verifyUserClusterUserKey"`
	}{}

	// Do the request
	err = p.GrapqhlRequest(`
		mutation ($clusterUserID:Int!, $key:String!) {
			manager_verifyUserClusterUserKey(
				clusterUserID: $clusterUserID,
				key: $key
			)
		}
	`, map[string]interface{}{
		"clusterUserID": clusterUser.ClusterUserID,
		"key":           key,
	}, &response)
	if err != nil {
		return err
	}

	return nil
}
