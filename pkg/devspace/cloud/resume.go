package cloud

import (
	"fmt"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
)

// ResumeSpace signals the cloud that we are currently working on the space and resumes it if it's currently paused
func ResumeSpace(config *latest.Config, generatedConfig *generated.Config, loop bool, log log.Logger) error {
	if generatedConfig.CloudSpace == nil {
		return nil
	}

	p, err := GetProvider(&generatedConfig.CloudSpace.ProviderName, log)
	if err != nil {
		return err
	}

	space, err := p.GetSpace(generatedConfig.CloudSpace.SpaceID)
	if err != nil {
		return fmt.Errorf("Error retrieving Spaces details: %v", err)
	}

	resumed, err := p.ResumeSpace(space.SpaceID, space.Cluster)
	if err != nil {
		return errors.Wrap(err, "active space")
	}

	// We will wait a little bit till the space has resumed
	if resumed {
		log.StartWait("Resuming space")
		defer log.StopWait()

		// Give the controllers some time to create the pods
		time.Sleep(time.Second * 3)

		// Create kubectl client and switch context if specified
		client, err := kubectl.NewClient(config)
		if err != nil {
			return fmt.Errorf("Unable to create new kubectl client: %v", err)
		}

		namespace, err := configutil.GetDefaultNamespace(config)
		if err != nil {
			return err
		}

		maxWait := time.Minute * 5
		start := time.Now()

		for time.Now().Sub(start) <= maxWait {
			pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
			if err != nil {
				return errors.Wrap(err, "list pods")
			}

			continueWaiting := false
			for _, pod := range pods.Items {
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if containerStatus.State.Waiting != nil {
						continueWaiting = true
					}
				}
			}

			if !continueWaiting {
				break
			}
		}
	}

	if loop {
		go func() {
			for {
				time.Sleep(time.Minute * 3)
				p.ResumeSpace(space.SpaceID, space.Cluster)
			}
		}()
	}

	return nil
}

// ResumeSpace resumes a space if its sleeping and sets the last activity to the current timestamp
func (p *Provider) ResumeSpace(spaceID int, cluster *Cluster) (bool, error) {
	key, err := p.GetClusterKey(cluster)
	if err != nil {
		return false, errors.Wrap(err, "get cluster key")
	}

	// Do the request
	response := &struct {
		ResumeSpace bool `json:"manager_resumeSpace"`
	}{}
	err = p.GrapqhlRequest(`
		mutation ($key:String, $spaceID: Int!){
			manager_resumeSpace(key: $key, spaceID: $spaceID)
		}
	`, map[string]interface{}{
		"key":     key,
		"spaceID": spaceID,
	}, response)
	if err != nil {
		return false, err
	}

	return response.ResumeSpace, nil
}
