package kaniko

import (
	"context"
	"fmt"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"io"
	"io/ioutil"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"

	"k8s.io/client-go/util/exec"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/restart"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/pullsecrets"
	"github.com/loft-sh/devspace/pkg/devspace/services"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/randutil"

	"os"
	"path/filepath"
	"time"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	dockerterm "github.com/docker/docker/pkg/term"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/util/interrupt"
)

// EngineName is the name of the building engine
const EngineName = "kaniko"

var (
	_, stdout, stderr = dockerterm.StdStreams()
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	PullSecretName string
	FullImageName  string
	BuildNamespace string

	allowInsecureRegistry bool
	dockerClient          docker.Client
}

// Wait timeout is the maximum time to wait for the kaniko init and build container to get ready
const waitTimeout = 20 * time.Minute

// NewBuilder creates a new kaniko.Builder instance
func NewBuilder(config *latest.Config, dockerClient docker.Client, kubeClient kubectl.Client, imageConfigName string, imageConf *latest.ImageConfig, imageTags []string, log logpkg.Logger) (builder.Interface, error) {
	buildNamespace := kubeClient.Namespace()
	if imageConf.Build.Kaniko.Namespace != "" {
		buildNamespace = imageConf.Build.Kaniko.Namespace
	}

	allowInsecurePush := false
	if imageConf.Build.Kaniko.Insecure != nil {
		allowInsecurePush = *imageConf.Build.Kaniko.Insecure
	}

	pullSecretName := ""
	if imageConf.Build.Kaniko.PullSecret != "" {
		pullSecretName = imageConf.Build.Kaniko.PullSecret
	}

	builder := &Builder{
		PullSecretName: pullSecretName,
		FullImageName:  imageConf.Image + ":" + imageTags[0],
		BuildNamespace: buildNamespace,

		allowInsecureRegistry: allowInsecurePush,

		dockerClient: dockerClient,
		helper:       helper.NewBuildHelper(config, kubeClient, EngineName, imageConfigName, imageConf, imageTags),
	}

	// create pull secret
	if !imageConf.Build.Kaniko.SkipPullSecretMount {
		err := builder.createPullSecret(log)
		if err != nil {
			return nil, errors.Wrap(err, "create pull secret")
		}
	}

	return builder, nil
}

// Build implements the interface
func (b *Builder) Build(log logpkg.Logger) error {
	return b.helper.Build(b, log)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(cache *generated.CacheConfig, forceRebuild, ignoreContextPathChanges bool) (bool, error) {
	return b.helper.ShouldRebuild(cache, forceRebuild, ignoreContextPathChanges)
}

// Authenticate authenticates kaniko for pushing to the RegistryURL (if username == "", it will try to get login data from local docker daemon)
func (b *Builder) createPullSecret(log logpkg.Logger) error {
	username, password := "", ""

	if b.PullSecretName != "" {
		return nil
	}

	registryURL, err := pullsecrets.GetRegistryFromImageName(b.FullImageName)
	if err != nil {
		return err
	}

	email := "noreply@devspace.cloud"
	authConfig, err := b.dockerClient.GetAuthConfig(registryURL, true)
	if err != nil {
		return err
	}

	username = authConfig.Username
	email = authConfig.Email

	if authConfig.Password != "" {
		password = authConfig.Password
	} else {
		password = authConfig.IdentityToken
	}

	return pullsecrets.NewClient(nil, nil, b.helper.KubeClient, b.dockerClient, log).CreatePullSecret(&pullsecrets.PullSecretOptions{
		Namespace:       b.BuildNamespace,
		RegistryURL:     registryURL,
		Username:        username,
		PasswordOrToken: password,
		Email:           email,
	})
}

// BuildImage builds a dockerimage within a kaniko pod
func (b *Builder) BuildImage(contextPath, dockerfilePath string, entrypoint []string, cmd []string, log logpkg.Logger) error {
	var err error

	// Buildoptions
	options := &types.ImageBuildOptions{}
	if b.helper.ImageConf.Build != nil && b.helper.ImageConf.Build.Kaniko != nil && b.helper.ImageConf.Build.Kaniko.Options != nil {
		if b.helper.ImageConf.Build.Kaniko.Options.BuildArgs != nil {
			options.BuildArgs = b.helper.ImageConf.Build.Kaniko.Options.BuildArgs
		}
		if b.helper.ImageConf.Build.Kaniko.Options.Target != "" {
			options.Target = b.helper.ImageConf.Build.Kaniko.Options.Target
		}
		if b.helper.ImageConf.Build.Kaniko.Options.Network != "" {
			options.NetworkMode = b.helper.ImageConf.Build.Kaniko.Options.Network
		}
	}

	// Check if we should overwrite entrypoint
	if len(entrypoint) > 0 || len(cmd) > 0 || b.helper.ImageConf.InjectRestartHelper || len(b.helper.ImageConf.AppendDockerfileInstructions) > 0 {
		dockerfilePath, err = helper.RewriteDockerfile(dockerfilePath, entrypoint, cmd, b.helper.ImageConf.AppendDockerfileInstructions, options.Target, b.helper.ImageConf.InjectRestartHelper, log)
		if err != nil {
			return err
		}

		defer os.RemoveAll(filepath.Dir(dockerfilePath))
	}

	// Generate the build pod spec
	randString := randutil.GenerateRandomString(12)
	buildID := strings.ToLower(randString)
	buildPod, err := b.getBuildPod(buildID, options, dockerfilePath)
	if err != nil {
		return errors.Wrap(err, "get build pod")
	}

	// Delete the build pod when we are done or get interrupted during build
	deleteBuildPod := func() {
		gracePeriod := int64(3)
		if buildPod.Name == "" {
			return
		}

		deleteErr := b.helper.KubeClient.KubeClient().CoreV1().Pods(b.BuildNamespace).Delete(context.TODO(), buildPod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})

		if deleteErr != nil {
			log.Errorf("Failed to delete build pod: %s", deleteErr.Error())
		}
	}

	intr := interrupt.New(nil, deleteBuildPod)
	err = intr.Run(func() error {
		defer log.StopWait()

		buildPodCreated, err := b.helper.KubeClient.KubeClient().CoreV1().Pods(b.BuildNamespace).Create(context.TODO(), buildPod, metav1.CreateOptions{})
		if err != nil {
			return errors.Errorf("unable to create build pod: %s", err.Error())
		}

		log.StartWait("Waiting for build init container to start")
		err = wait.PollImmediate(time.Second, waitTimeout, func() (done bool, err error) {
			buildPod, err = b.helper.KubeClient.KubeClient().CoreV1().Pods(b.BuildNamespace).Get(context.TODO(), buildPodCreated.Name, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			} else if len(buildPod.Status.InitContainerStatuses) > 0 {
				status := buildPod.Status.InitContainerStatuses[0]
				if status.State.Terminated != nil {
					errorLog := ""
					reader, _ := b.helper.KubeClient.Logs(context.TODO(), b.BuildNamespace, buildPodCreated.Name, buildPod.Spec.InitContainers[0].Name, false, nil, false)
					if reader != nil {
						out, err := ioutil.ReadAll(reader)
						if err == nil {
							errorLog = string(out)
						}
					}
					if errorLog == "" {
						errorLog = buildPod.Status.InitContainerStatuses[0].State.Terminated.Message
					}

					return false, fmt.Errorf("kaniko init container %s/%s has unexpectedly exited with code %d: %s", buildPod.Namespace, buildPod.Name, buildPod.Status.InitContainerStatuses[0].State.Terminated.ExitCode, errorLog)
				} else if status.State.Waiting != nil {
					if kubectl.CriticalStatus[status.State.Waiting.Reason] {
						return false, fmt.Errorf("kaniko init container %s/%s cannot start: %s (%s)", buildPod.Namespace, buildPod.Name, status.State.Waiting.Message, status.State.Waiting.Reason)
					}
				}
			}

			return len(buildPod.Status.InitContainerStatuses) > 0 && buildPod.Status.InitContainerStatuses[0].State.Running != nil, nil
		})
		if err != nil {
			return errors.Wrap(err, "waiting for kaniko init")
		}

		// Get ignore rules from docker ignore
		ignoreRules, err := helper.ReadDockerignore(contextPath)
		if err != nil {
			return err
		}
		if err := build.ValidateContextDirectory(contextPath, ignoreRules); err != nil {
			return errors.Errorf("error checking context: '%s'", err)
		}
		relDockerfile := archive.CanonicalTarNameForPath(dockerfilePath)
		ignoreRules = build.TrimBuildFilesFromExcludes(ignoreRules, relDockerfile, false)
		ignoreRules = append(ignoreRules, ".devspace/")

		log.StartWait("Uploading files to build container")
		buildCtx, err := archive.TarWithOptions(contextPath, &archive.TarOptions{
			ExcludePatterns: ignoreRules,
			ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
		})
		if err != nil {
			return err
		}

		// Copy complete context
		_, stderr, err := b.helper.KubeClient.ExecBuffered(buildPod, buildPod.Spec.InitContainers[0].Name, []string{"tar", "xp", "-C", kanikoContextPath + "/."}, buildCtx)
		if err != nil {
			if stderr != nil {
				return errors.Errorf("copy context: error executing tar: %s: %v", string(stderr), err)
			}

			return errors.Wrap(err, "copy context")
		}

		// Copy dockerfile
		err = b.helper.KubeClient.Copy(buildPod, buildPod.Spec.InitContainers[0].Name, kanikoContextPath, dockerfilePath, []string{})
		if err != nil {
			return errors.Errorf("error uploading dockerfile to container: %v", err)
		}

		// Copy restart helper script
		if b.helper.ImageConf.InjectRestartHelper {
			tempDir, err := ioutil.TempDir("", "")
			if err != nil {
				return err
			}

			defer os.RemoveAll(tempDir)

			scriptPath := filepath.Join(tempDir, restart.ScriptName)
			remoteFolder := filepath.ToSlash(filepath.Join(kanikoContextPath, ".devspace", ".devspace"))
			helperScript, err := restart.LoadRestartHelper(b.helper.ImageConf.RestartHelperPath)
			if err != nil {
				return errors.Wrap(err, "load restart helper")
			}

			err = ioutil.WriteFile(scriptPath, []byte(helperScript), 0777)
			if err != nil {
				return errors.Wrap(err, "write restart helper script")
			}

			// create the .devspace directory in the container
			_, _, err = b.helper.KubeClient.ExecBuffered(buildPod, buildPod.Spec.InitContainers[0].Name, []string{"mkdir", "-p", remoteFolder}, nil)
			if err != nil {
				return errors.Errorf("error executing command 'mkdir -p %s' in init container: %v", remoteFolder, err)
			}

			// copy the helper script into the container
			err = b.helper.KubeClient.Copy(buildPod, buildPod.Spec.InitContainers[0].Name, remoteFolder, scriptPath, []string{})
			if err != nil {
				return errors.Errorf("error uploading helper script to container: %v", err)
			}

			// change permissions for the execution script
			_, _, err = b.helper.KubeClient.ExecBuffered(buildPod, buildPod.Spec.InitContainers[0].Name, []string{"chmod", "-R", "0777", remoteFolder}, nil)
			if err != nil {
				return errors.Errorf("error executing command 'chmod +x %s' in init container: %v", filepath.Join(kanikoContextPath, restart.ScriptName), err)
			}

			// remove the .dockerignore since .devspace is usually ignored and we want to sneak our helper script in
			// this shouldn't be any issue since the context was already pruned in the copy step beforehand
			_, _, err = b.helper.KubeClient.ExecBuffered(buildPod, buildPod.Spec.InitContainers[0].Name, []string{"rm", filepath.ToSlash(filepath.Join(kanikoContextPath, ".dockerignore"))}, nil)
			if err != nil {
				if _, ok := err.(exec.CodeExitError); !ok {
					return errors.Errorf("error executing command 'rm .dockerignore' in init container: %v", err)
				}
			}
		}

		// Tell init container we are done
		_, _, err = b.helper.KubeClient.ExecBuffered(buildPod, buildPod.Spec.InitContainers[0].Name, []string{"touch", doneFile}, nil)
		if err != nil {
			return errors.Errorf("Error executing command in init container: %v", err)
		}

		log.Done("Uploaded files to container")
		log.StartWait("Waiting for kaniko container to start")
		err = wait.PollImmediate(time.Second, waitTimeout, func() (done bool, err error) {
			buildPod, err = b.helper.KubeClient.KubeClient().CoreV1().Pods(b.BuildNamespace).Get(context.TODO(), buildPodCreated.Name, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			} else if len(buildPod.Status.ContainerStatuses) > 0 {
				status := buildPod.Status.ContainerStatuses[0]
				if status.State.Terminated != nil {
					errorLog := ""
					reader, _ := b.helper.KubeClient.Logs(context.TODO(), b.BuildNamespace, buildPodCreated.Name, status.Name, false, nil, false)
					if reader != nil {
						out, err := ioutil.ReadAll(reader)
						if err == nil {
							errorLog = string(out)
						}
					}
					if errorLog == "" {
						errorLog = buildPod.Status.ContainerStatuses[0].State.Terminated.Message
					}

					return false, fmt.Errorf("kaniko pod %s/%s has unexpectedly exited with code %d: %s", buildPod.Namespace, buildPod.Name, status.State.Terminated.ExitCode, errorLog)
				} else if status.State.Waiting != nil {
					if kubectl.CriticalStatus[status.State.Waiting.Reason] {
						return false, fmt.Errorf("kaniko pod %s/%s cannot start: %s (%s)", buildPod.Namespace, buildPod.Name, status.State.Waiting.Message, status.State.Waiting.Reason)
					}
				}
			}

			return len(buildPod.Status.ContainerStatuses) > 0 && buildPod.Status.ContainerStatuses[0].Ready, nil
		})
		if err != nil {
			return errors.Wrap(err, "waiting for kaniko")
		}

		log.StopWait()
		log.Done("Build pod has started")

		// Determine output writer
		var writer io.Writer
		if log == logpkg.GetInstance() {
			writer = stdout
		} else {
			writer = log
		}

		stdoutLogger := kanikoLogger{out: writer}

		// Stream the logs
		err = services.NewClient(b.helper.Config, nil, b.helper.KubeClient, log).StartLogsWithWriter(targetselector.NewOptionsFromFlags(buildPod.Spec.Containers[0].Name, "", buildPod.Namespace, buildPod.Name, false), true, 100, false, stdoutLogger)
		if err != nil {
			return errors.Errorf("error printing build logs: %v", err)
		}

		log.StartWait("Checking build status")
		for true {
			time.Sleep(time.Second)

			// Check if build was successful
			pod, err := b.helper.KubeClient.KubeClient().CoreV1().Pods(b.BuildNamespace).Get(context.TODO(), buildPodCreated.Name, metav1.GetOptions{})
			if err != nil {
				return errors.Errorf("Error checking if build was successful: %v", err)
			}

			// Check if terminated
			if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].State.Terminated != nil {
				if pod.Status.ContainerStatuses[0].State.Terminated.ExitCode != 0 {
					return errors.Errorf("error building image (Exit Code %d)", pod.Status.ContainerStatuses[0].State.Terminated.ExitCode)
				}

				break
			}
		}
		log.StopWait()

		log.Done("Done building image")
		return nil
	})
	if err != nil {
		// Delete all build pods on error
		pods, getErr := b.helper.KubeClient.KubeClient().CoreV1().Pods(b.BuildNamespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: "devspace-build=true",
		})
		if getErr != nil {
			return err
		}
		for _, pod := range pods.Items {
			b.helper.KubeClient.KubeClient().CoreV1().Pods(b.BuildNamespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
		}

		return err
	}

	return nil
}
