package cloud

import (
	"time"

	cloudlatest "github.com/devspace-cloud/devspace/pkg/devspace/cloud/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
)

// ResumeSpace signals the cloud that we are currently working on the space and resumes it if it's currently paused
func ResumeSpace(client kubectl.Client, loop bool, log log.Logger) error {
	isSpace, err := kubeconfig.IsCloudSpace(client.CurrentContext())
	if err != nil {
		return errors.Wrap(err, "is cloud space")
	}

	// It is not a space so we just exit here
	if isSpace == false {
		return nil
	}

	// Retrieve space id and cloud provider
	spaceID, cloudProvider, err := kubeconfig.GetSpaceID(client.CurrentContext())
	if err != nil {
		return errors.Errorf("Unable to get Space ID for context '%s': %v", client.CurrentContext(), err)
	}

	p, err := GetProvider(cloudProvider, log)
	if err != nil {
		return err
	}

	// Retrieve space from cache
	space, _, err := p.GetAndUpdateSpaceCache(spaceID, false)
	if err != nil {
		return err
	}

	resumed, err := p.ResumeSpace(spaceID, space.Space.Cluster)
	if err != nil {
		return errors.Wrap(err, "resume space")
	}

	// We will wait a little bit till the space has resumed
	if resumed {
		log.StartWait("Resuming space")
		defer log.StopWait()

		// Give the controllers some time to create the pods
		time.Sleep(time.Second * 3)

		err = WaitForSpaceResume(client)
		if err != nil {
			return err
		}
	}

	if loop {
		go func() {
			for {
				time.Sleep(time.Minute * 3)
				p.ResumeSpace(spaceID, space.Space.Cluster)
			}
		}()
	}

	return nil
}

// WaitForSpaceResume waits for a space to resume
func WaitForSpaceResume(client kubectl.Client) error {
	maxWait := time.Minute * 5
	start := time.Now()

	for time.Now().Sub(start) <= maxWait {
		pods, err := client.KubeClient().CoreV1().Pods(client.Namespace()).List(metav1.ListOptions{})
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

		time.Sleep(1 * time.Second)
	}

	return nil
}

// ResumeSpace resumes a space if its sleeping and sets the last activity to the current timestamp
func (p *Provider) ResumeSpace(spaceID int, cluster *cloudlatest.Cluster) (bool, error) {
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
