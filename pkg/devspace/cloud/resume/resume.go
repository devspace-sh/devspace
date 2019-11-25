package resume

import (
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/cloud"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
)

//SpaceResumer can resume a space
type SpaceResumer interface {
	ResumeSpace(loop bool) error
}

type resumer struct {
	kubeClient kubectl.Client
	log        log.Logger
}

// NewSpaceResumer creates a new instance of the interface SpaceResumer
func NewSpaceResumer(kubeClient kubectl.Client, log log.Logger) SpaceResumer {
	return &resumer{
		kubeClient: kubeClient,
		log:        log,
	}
}

// ResumeSpace signals the cloud that we are currently working on the space and resumes it if it's currently paused
func (r *resumer) ResumeSpace(loop bool) error {
	isSpace, err := kubeconfig.IsCloudSpace(r.kubeClient.CurrentContext())
	if err != nil {
		return errors.Wrap(err, "is cloud space")
	}

	// It is not a space so we just exit here
	if isSpace == false {
		return nil
	}

	// Retrieve space id and cloud provider
	spaceID, cloudProvider, err := kubeconfig.GetSpaceID(r.kubeClient.CurrentContext())
	if err != nil {
		return errors.Errorf("Unable to get Space ID for context '%s': %v", r.kubeClient.CurrentContext(), err)
	}

	p, err := cloud.GetProvider(cloudProvider, r.log)
	if err != nil {
		return err
	}

	// Retrieve space from cache
	space, _, err := p.GetAndUpdateSpaceCache(spaceID, false)
	if err != nil {
		return err
	}

	key, err := p.GetClusterKey(space.Space.Cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster key")
	}

	resumed, err := p.Client().ResumeSpace(spaceID, key, space.Space.Cluster)
	if err != nil {
		return errors.Wrap(err, "resume space")
	}

	// We will wait a little bit till the space has resumed
	if resumed {
		r.log.StartWait("Resuming space")
		defer r.log.StopWait()

		// Give the controllers some time to create the pods
		time.Sleep(time.Second * 3)

		err = r.waitForSpaceResume()
		if err != nil {
			return err
		}
	}

	if loop {
		go func() {
			for {
				time.Sleep(time.Minute * 3)
				p.Client().ResumeSpace(spaceID, key, space.Space.Cluster)
			}
		}()
	}

	return nil
}

// WaitForSpaceResume waits for a space to resume
func (r *resumer) waitForSpaceResume() error {
	maxWait := time.Minute * 5
	start := time.Now()

	for time.Now().Sub(start) <= maxWait {
		pods, err := r.kubeClient.KubeClient().CoreV1().Pods(r.kubeClient.Namespace()).List(metav1.ListOptions{})
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
