package kaniko

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/builder"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/registry"
	"github.com/devspace-cloud/devspace/pkg/devspace/services"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/devspace-cloud/devspace/pkg/devspace/sync"
	"github.com/devspace-cloud/devspace/pkg/util/ignoreutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	dockerterm "github.com/docker/docker/pkg/term"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/util/interrupt"
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	PullSecretName string
	ImageName      string
	BuildNamespace string

	allowInsecureRegistry bool
	kanikoOptions         *latest.KanikoConfig
	kubectl               *kubernetes.Clientset
	dockerClient          client.CommonAPIClient
}

// Wait timeout is the maximum time to wait for the kaniko init and build container to get ready
const waitTimeout = 2 * time.Minute

// NewBuilder creates a new kaniko.Builder instance
func NewBuilder(pullSecretName, imageName, imageTag, buildNamespace string, kanikoOptions *latest.KanikoConfig, dockerClient client.CommonAPIClient, kubectl *kubernetes.Clientset, allowInsecureRegistry bool) (*Builder, error) {
	return &Builder{
		PullSecretName: pullSecretName,
		ImageName:      imageName + ":" + imageTag,
		BuildNamespace: buildNamespace,

		allowInsecureRegistry: allowInsecureRegistry,
		kanikoOptions:         kanikoOptions,
		kubectl:               kubectl,
		dockerClient:          dockerClient,
	}, nil
}

// Authenticate authenticates kaniko for pushing to the RegistryURL (if username == "", it will try to get login data from local docker daemon)
func (b *Builder) Authenticate() (*types.AuthConfig, error) {
	username, password := "", ""

	if b.PullSecretName != "" {
		return nil, nil
	}

	registryURL, err := registry.GetRegistryFromImageName(b.ImageName)
	if err != nil {
		return nil, err
	}

	email := "noreply@devspace.cloud"
	authConfig, err := docker.GetAuthConfig(b.dockerClient, registryURL, true)
	if err != nil {
		return nil, err
	}

	username = authConfig.Username
	email = authConfig.Email

	if authConfig.Password != "" {
		password = authConfig.Password
	} else {
		password = authConfig.IdentityToken
	}

	return nil, registry.CreatePullSecret(b.kubectl, b.BuildNamespace, registryURL, username, password, email, log.GetInstance())
}

// BuildImage builds a dockerimage within a kaniko pod
func (b *Builder) BuildImage(contextPath, dockerfilePath string, options *types.ImageBuildOptions, entrypoint *[]*string) error {
	var err error

	// Check if we should overwrite entrypoint
	if entrypoint != nil && len(*entrypoint) > 0 {
		dockerfilePath, err = builder.CreateTempDockerfile(dockerfilePath, *entrypoint)
		if err != nil {
			return err
		}

		defer os.RemoveAll(filepath.Dir(dockerfilePath))
	}

	// Generate the build pod spec
	buildPod, err := b.getBuildPod(options, dockerfilePath)
	if err != nil {
		return errors.Wrap(err, "get build pod")
	}

	// Delete the build pod when we are done or get interrupted during build
	deleteBuildPod := func() {
		gracePeriod := int64(3)

		deleteErr := b.kubectl.Core().Pods(b.BuildNamespace).Delete(buildPod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})

		if deleteErr != nil {
			log.Errorf("Failed to delete build pod: %s", deleteErr.Error())
		}
	}

	intr := interrupt.New(nil, deleteBuildPod)
	err = intr.Run(func() error {
		defer log.StopWait()

		pods, err := b.kubectl.Core().Pods(b.BuildNamespace).List(metav1.ListOptions{
			LabelSelector: "devspace-build=true",
		})
		if err != nil {
			return errors.Wrap(err, "list pods in build namespace")
		}

		if len(pods.Items) > 0 {
			log.StartWait("Deleting old build pods")

			for _, pod := range pods.Items {
				// Delete older build pods when they already exist
				err := b.kubectl.Core().Pods(b.BuildNamespace).Delete(pod.Name, &metav1.DeleteOptions{
					GracePeriodSeconds: ptr.Int64(0),
				})
				if err != nil {
					return errors.Wrap(err, "delete build pod")
				}
			}

			// Wait till all pods are deleted
			for true {
				time.Sleep(time.Second * 3)

				pods, err := b.kubectl.Core().Pods(b.BuildNamespace).List(metav1.ListOptions{
					LabelSelector: "devspace-build=true",
				})
				if err != nil || len(pods.Items) == 0 {
					break
				}
			}
		}

		buildPodCreated, err := b.kubectl.Core().Pods(b.BuildNamespace).Create(buildPod)
		if err != nil {
			return fmt.Errorf("Unable to create build pod: %s", err.Error())
		}

		now := time.Now()
		log.StartWait("Waiting for build init container to start")

		for {
			buildPod, _ = b.kubectl.Core().Pods(b.BuildNamespace).Get(buildPodCreated.Name, metav1.GetOptions{})
			if len(buildPod.Status.InitContainerStatuses) > 0 && buildPod.Status.InitContainerStatuses[0].State.Running != nil {
				break
			}

			time.Sleep(5 * time.Second)
			if time.Since(now) >= waitTimeout {
				return fmt.Errorf("Timeout waiting for init container")
			}
		}

		// Get ignore rules from docker ignore
		ignoreRules, ignoreRuleErr := ignoreutil.GetIgnoreRules(contextPath)
		if ignoreRuleErr != nil {
			return fmt.Errorf("Unable to parse .dockerignore files: %s", ignoreRuleErr.Error())
		}

		log.StartWait("Uploading files to build container")

		// Copy complete context
		err = sync.CopyToContainer(b.kubectl, buildPod, &buildPod.Spec.InitContainers[0], contextPath, kanikoContextPath, ignoreRules)
		if err != nil {
			return fmt.Errorf("Error uploading files to container: %v", err)
		}

		// Copy dockerfile
		err = sync.CopyToContainer(b.kubectl, buildPod, &buildPod.Spec.InitContainers[0], dockerfilePath, kanikoContextPath, ignoreRules)
		if err != nil {
			return fmt.Errorf("Error uploading files to container: %v", err)
		}

		// Tell init container we are done
		_, _, err = kubectl.ExecBuffered(b.kubectl, buildPod, buildPod.Spec.InitContainers[0].Name, []string{"touch", doneFile})
		if err != nil {
			return fmt.Errorf("Error executing command in init container: %v", err)
		}

		log.Done("Uploaded files to container")
		log.StartWait("Waiting for kaniko container to start")

		now = time.Now()
		for true {
			buildPod, _ = b.kubectl.Core().Pods(b.BuildNamespace).Get(buildPodCreated.Name, metav1.GetOptions{})
			if len(buildPod.Status.ContainerStatuses) > 0 && buildPod.Status.ContainerStatuses[0].Ready {
				break
			}

			time.Sleep(2 * time.Second)
			if time.Since(now) >= waitTimeout {
				return fmt.Errorf("Timeout waiting for kaniko build pod")
			}
		}

		log.StopWait()
		log.Done("Build pod has started")

		_, stdout, stderr := dockerterm.StdStreams()
		stdoutLogger := kanikoLogger{out: stdout}
		stderrLogger := kanikoLogger{out: stderr}

		// Stream the logs
		err = services.StartLogsWithWriter(b.kubectl, targetselector.CmdParameter{PodName: &buildPod.Name, ContainerName: &buildPod.Spec.Containers[0].Name, Namespace: &buildPod.Namespace}, true, 100, log.GetInstance(), stdoutLogger, stderrLogger)
		if err != nil {
			return fmt.Errorf("Error during printling build logs: %v", err)
		}

		log.StartWait("Checking build status")
		for true {
			time.Sleep(time.Second)

			// Check if build was successfull
			pod, err := b.kubectl.Core().Pods(b.BuildNamespace).Get(buildPodCreated.Name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("Error checking if build was successful: %v", err)
			}

			// Check if terminated
			if pod.Status.ContainerStatuses[0].State.Terminated != nil {
				if pod.Status.ContainerStatuses[0].State.Terminated.ExitCode != 0 {
					return fmt.Errorf("Error building image (Exit Code %d)", pod.Status.ContainerStatuses[0].State.Terminated.ExitCode)
				}

				break
			}
		}
		log.StopWait()

		log.Done("Done building image")
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// PushImage is required to implement builder.Interface
func (b *Builder) PushImage() error {
	return nil
}
