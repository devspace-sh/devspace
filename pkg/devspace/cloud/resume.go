package cloud

import (
	"fmt"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"
)

// ResumeLatestSpace resumes the latest Space that has been used to deploy this project to (if any)
func ResumeLatestSpace(config *latest.Config, loop bool, log log.Logger) error {
	generatedConfig, err := generated.LoadConfig()
	if err != nil {
		return err
	}

	if generatedConfig != nil && generatedConfig.Namespace != nil && generatedConfig.Namespace.KubeContext != nil {
		context, contextName, err := kubeconfig.GetContext(*generatedConfig.Namespace.KubeContext)
		if err != nil {
			return fmt.Errorf("Unable to get current kube-context: %v", err)
		}

		spaceID, cloudProvider, err := kubeconfig.GetSpaceID(context)
		if err != nil {
			return fmt.Errorf("Unable to get Space ID for context %s: %v", contextName, err)
		}

		return ResumeSpace(config, cloudProvider, spaceID, loop, log)
	}
	return nil
}

// ResumeSpace signals the cloud that we are currently working on the space and resumes it if it's currently paused
func ResumeSpace(config *latest.Config, cloudProvider string, spaceID int, loop bool, log log.Logger) error {
	p, err := GetProvider(&cloudProvider, log)
	if err != nil {
		return err
	}

	space, err := p.GetSpace(spaceID)
	if err != nil {
		return fmt.Errorf("Error retrieving Spaces details: %v", err)
	}

	resumed, err := p.ResumeSpace(space.SpaceID, space.Cluster)
	if err != nil {
		return errors.Wrap(err, "resume space")
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

		err = WaitForSpaceResume(client, namespace)
		if err != nil {
			return err
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

// WaitForSpaceResume waits for a space to resume
func WaitForSpaceResume(client kubernetes.Interface, namespace string) error {
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

		time.Sleep(1 * time.Second)
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
