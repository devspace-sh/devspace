package cloud

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"
)

// GetClusterKey makes sure there is a correct key for the given cluster id
func (p *provider) GetClusterKey(cluster *latest.Cluster) (string, error) {
	if cluster.EncryptToken == false {
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
			return p.askForEncryptionKey(cluster)
		}
	}

	// Verifies the cluster key
	verified, err := p.client.VerifyKey(cluster.ClusterID, key)
	if err != nil {
		return "", errors.Wrap(err, "verify key")
	}
	if verified == false {
		return p.askForEncryptionKey(cluster)
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

// askForEncryptionKey asks the user for his her encryption key and verifies that the key is correct
func (p *provider) askForEncryptionKey(cluster *latest.Cluster) (string, error) {
	p.log.StopWait()

	// Wait till user enters the correct key
	for true {
		key, err := p.log.Question(&survey.QuestionOptions{
			Question:               "Please enter your encryption key for cluster " + cluster.Name,
			ValidationRegexPattern: "^.{6,32}$",
			ValidationMessage:      "Key has to be between 6 and 32 characters long",
			IsPassword:             true,
		})
		if err != nil {
			return "", err
		}

		hashedKey, err := hash.Password(key)
		if err != nil {
			return "", errors.Wrap(err, "hash key")
		}

		verified, err := p.client.VerifyKey(cluster.ClusterID, hashedKey)
		if err != nil {
			return "", errors.Wrap(err, "verify key")
		}
		if verified == false {
			p.log.Errorf("Encryption key is incorrect. Please try again")
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
