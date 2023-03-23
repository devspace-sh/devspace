package kaniko

import (
	"fmt"
	"io"
	"strings"

	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	"github.com/loft-sh/devspace/pkg/devspace/services/logs"
	"github.com/sirupsen/logrus"

	"github.com/loft-sh/devspace/pkg/util/interrupt"
	"github.com/loft-sh/devspace/pkg/util/progressreader"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/exec"

	"github.com/loft-sh/devspace/pkg/devspace/build/builder"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/restart"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/randutil"

	"os"
	"path/filepath"
	"time"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	dockerterm "github.com/moby/term"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EngineName is the name of the building engine
const EngineName = "kaniko"

var (
	_, stdout, _ = dockerterm.StdStreams()
)

// Builder holds the necessary information to build and push docker images
type Builder struct {
	helper *helper.BuildHelper

	PullSecretName string
	FullImageName  string
	BuildNamespace string

	allowInsecureRegistry bool
}

// Wait timeout is the maximum time to wait for the kaniko init and build container to get ready
const waitTimeout = 20 * time.Minute

// NewBuilder creates a new kaniko.Builder instance
func NewBuilder(ctx devspacecontext.Context, imageConf *latest.Image, imageTags []string) (builder.Interface, error) {
	if imageConf.Kaniko != nil && imageConf.Kaniko.Namespace != "" {
		err := kubectl.EnsureNamespace(ctx.Context(), ctx.KubeClient(), imageConf.Kaniko.Namespace, ctx.Log())
		if err != nil {
			return nil, err
		}
	}

	buildNamespace := ctx.KubeClient().Namespace()
	if imageConf.Kaniko.Namespace != "" {
		buildNamespace = imageConf.Kaniko.Namespace
	}

	allowInsecurePush := false
	if imageConf.Kaniko.Insecure != nil {
		allowInsecurePush = *imageConf.Kaniko.Insecure
	}

	pullSecretName := ""
	if imageConf.Kaniko.PullSecret != "" {
		pullSecretName = imageConf.Kaniko.PullSecret
	}

	b := &Builder{
		PullSecretName: pullSecretName,
		FullImageName:  imageConf.Image + ":" + imageTags[0],
		BuildNamespace: buildNamespace,

		allowInsecureRegistry: allowInsecurePush,
		helper:                helper.NewBuildHelper(ctx, EngineName, imageConf, imageTags),
	}

	return b, nil
}

// Build implements the interface
func (b *Builder) Build(ctx devspacecontext.Context) error {
	return b.helper.Build(ctx, b)
}

// ShouldRebuild determines if an image has to be rebuilt
func (b *Builder) ShouldRebuild(ctx devspacecontext.Context, forceRebuild bool) (bool, error) {
	return b.helper.ShouldRebuild(ctx, forceRebuild)
}

// BuildImage builds a dockerimage within a kaniko pod
func (b *Builder) BuildImage(ctx devspacecontext.Context, contextPath, dockerfilePath string, entrypoint []string, cmd []string) error {
	var err error

	contextPath, err = build.ResolveAndValidateContextPath(contextPath)
	if err != nil {
		return errors.Wrap(err, "resolve context path")
	}

	// build options
	options := &types.ImageBuildOptions{}
	if b.helper.ImageConf.BuildArgs != nil {
		options.BuildArgs = b.helper.ImageConf.BuildArgs
	}
	if b.helper.ImageConf.Target != "" {
		options.Target = b.helper.ImageConf.Target
	}
	if b.helper.ImageConf.Network != "" {
		options.NetworkMode = b.helper.ImageConf.Network
	}

	// Check if we should overwrite entrypoint
	injectRestartHelper := b.helper.ImageConf.InjectRestartHelper || b.helper.ImageConf.InjectLegacyRestartHelper
	if len(entrypoint) > 0 || len(cmd) > 0 || injectRestartHelper || len(b.helper.ImageConf.AppendDockerfileInstructions) > 0 {
		dockerfilePath, err = helper.RewriteDockerfile(dockerfilePath, entrypoint, cmd, b.helper.ImageConf.AppendDockerfileInstructions, options.Target, injectRestartHelper, ctx.Log())
		if err != nil {
			return err
		}

		defer os.RemoveAll(filepath.Dir(dockerfilePath))
	}

	// Generate the build pod spec
	randString := randutil.GenerateRandomString(12)
	buildID := strings.ToLower(randString)
	buildPod, err := b.getBuildPod(ctx, buildID, options, dockerfilePath)
	if err != nil {
		return errors.Wrap(err, "get build pod")
	}

	// Delete the build pod when we are done or get interrupted during build
	deleteBuildPod := func() {
		gracePeriod := int64(3)
		if buildPod.Name == "" {
			return
		}

		deleteErr := ctx.KubeClient().KubeClient().CoreV1().Pods(b.BuildNamespace).Delete(ctx.Context(), buildPod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})

		if deleteErr != nil {
			ctx.Log().Errorf("Failed to delete build pod: %s", deleteErr.Error())
		}
	}

	err = interrupt.Global.RunAlways(func() error {
		buildPodCreated, err := ctx.KubeClient().KubeClient().CoreV1().Pods(b.BuildNamespace).Create(ctx.Context(), buildPod, metav1.CreateOptions{})
		if err != nil {
			return errors.Errorf("unable to create build pod: %s", err.Error())
		}

		ctx.Log().Info("Waiting for build init container to start...")
		err = wait.PollImmediate(time.Second, waitTimeout, func() (done bool, err error) {
			buildPod, err = ctx.KubeClient().KubeClient().CoreV1().Pods(b.BuildNamespace).Get(ctx.Context(), buildPodCreated.Name, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			} else if len(buildPod.Status.InitContainerStatuses) > 0 {
				status := buildPod.Status.InitContainerStatuses[0]
				if status.State.Terminated != nil {
					errorLog := ""
					reader, _ := ctx.KubeClient().Logs(ctx.Context(), b.BuildNamespace, buildPodCreated.Name, buildPod.Spec.InitContainers[0].Name, false, nil, false)
					if reader != nil {
						out, err := io.ReadAll(reader)
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
		relDockerfile := archive.CanonicalTarNameForPath(dockerfilePath)
		ignoreRules, err := helper.ReadDockerignore(contextPath, relDockerfile)
		if err != nil {
			return err
		}
		if err := build.ValidateContextDirectory(contextPath, ignoreRules); err != nil {
			return errors.Errorf("error checking context: '%s'", err)
		}

		ctx.Log().Info("Uploading files to build container...")
		buildCtx, err := archive.TarWithOptions(contextPath, &archive.TarOptions{
			ExcludePatterns: ignoreRules,
			ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
		})
		if err != nil {
			return err
		}

		// Wrap it with our custom io.ReadCloser in order to show progress.
		buildCtx = &progressreader.ProgressReader{ReadCloser: buildCtx, Ctx: ctx}

		// Copy complete context
		_, stderr, err := ctx.KubeClient().ExecBuffered(ctx.Context(), buildPod, buildPod.Spec.InitContainers[0].Name, []string{"tar", "xp", "-C", kanikoContextPath + "/."}, buildCtx)
		if err != nil {
			if stderr != nil {
				return errors.Errorf("copy context: error executing tar: %s: %v", string(stderr), err)
			}

			return errors.Wrap(err, "copy context")
		}

		// Copy dockerfile
		err = ctx.KubeClient().Copy(ctx.Context(), buildPod, buildPod.Spec.InitContainers[0].Name, kanikoContextPath, dockerfilePath, []string{})
		if err != nil {
			return errors.Errorf("error uploading dockerfile to container: %v", err)
		}

		// Copy restart helper script
		if injectRestartHelper {
			tempDir, err := os.MkdirTemp("", "")
			if err != nil {
				return err
			}

			defer os.RemoveAll(tempDir)

			scriptPath := filepath.Join(tempDir, restart.ScriptName)
			remoteFolder := filepath.ToSlash(filepath.Join(kanikoContextPath, ".devspace", ".devspace"))

			var helperScript string
			if b.helper.ImageConf.InjectRestartHelper {
				helperScript, err = restart.LoadRestartHelper(b.helper.ImageConf.RestartHelperPath)
				if err != nil {
					return errors.Wrap(err, "load restart helper")
				}
			} else if b.helper.ImageConf.InjectLegacyRestartHelper {
				helperScript, err = restart.LoadLegacyRestartHelper(b.helper.ImageConf.RestartHelperPath)
				if err != nil {
					return errors.Wrap(err, "load legacy restart helper")
				}
			}

			err = os.WriteFile(scriptPath, []byte(helperScript), 0777)
			if err != nil {
				return errors.Wrap(err, "write restart helper script")
			}

			// create the .devspace directory in the container
			_, _, err = ctx.KubeClient().ExecBuffered(ctx.Context(), buildPod, buildPod.Spec.InitContainers[0].Name, []string{"mkdir", "-p", remoteFolder}, nil)
			if err != nil {
				return errors.Errorf("error executing command 'mkdir -p %s' in init container: %v", remoteFolder, err)
			}

			// copy the helper script into the container
			err = ctx.KubeClient().Copy(ctx.Context(), buildPod, buildPod.Spec.InitContainers[0].Name, remoteFolder, scriptPath, []string{})
			if err != nil {
				return errors.Errorf("error uploading helper script to container: %v", err)
			}

			// change permissions for the execution script
			_, _, err = ctx.KubeClient().ExecBuffered(ctx.Context(), buildPod, buildPod.Spec.InitContainers[0].Name, []string{"chmod", "-R", "0777", remoteFolder}, nil)
			if err != nil {
				return errors.Errorf("error executing command 'chmod +x %s' in init container: %v", filepath.Join(kanikoContextPath, restart.ScriptName), err)
			}

			// remove the .dockerignore since .devspace is usually ignored and we want to sneak our helper script in
			// this shouldn't be any issue since the context was already pruned in the copy step beforehand
			_, _, err = ctx.KubeClient().ExecBuffered(ctx.Context(), buildPod, buildPod.Spec.InitContainers[0].Name, []string{"rm", filepath.ToSlash(filepath.Join(kanikoContextPath, ".dockerignore"))}, nil)
			if err != nil {
				if _, ok := err.(exec.CodeExitError); !ok {
					return errors.Errorf("error executing command 'rm .dockerignore' in init container: %v", err)
				}
			}
		}

		// Tell init container we are done
		_, _, err = ctx.KubeClient().ExecBuffered(ctx.Context(), buildPod, buildPod.Spec.InitContainers[0].Name, []string{"touch", doneFile}, nil)
		if err != nil {
			return errors.Errorf("Error executing command in init container: %v", err)
		}

		ctx.Log().Done("Uploaded files to container")
		ctx.Log().Info("Waiting for kaniko container to start...")
		err = wait.PollImmediate(time.Second, waitTimeout, func() (done bool, err error) {
			buildPod, err = ctx.KubeClient().KubeClient().CoreV1().Pods(b.BuildNamespace).Get(ctx.Context(), buildPodCreated.Name, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			} else if len(buildPod.Status.ContainerStatuses) > 0 {
				status := buildPod.Status.ContainerStatuses[0]
				if status.State.Terminated != nil {
					if status.State.Terminated.ExitCode == 0 {
						return true, nil
					}

					errorLog := ""
					reader, _ := ctx.KubeClient().Logs(ctx.Context(), b.BuildNamespace, buildPodCreated.Name, status.Name, false, nil, false)
					if reader != nil {
						out, err := io.ReadAll(reader)
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

		ctx.Log().Done("Build pod has started")

		// Determine output writer
		var writer io.WriteCloser
		if ctx.Log() == logpkg.GetInstance() {
			writer = logpkg.WithNopCloser(stdout)
		} else {
			writer = ctx.Log().Writer(logrus.InfoLevel, false)
		}
		defer writer.Close()

		stdoutLogger := kanikoLogger{out: writer}

		// Stream the logs
		options := targetselector.NewOptionsFromFlags(buildPod.Spec.Containers[0].Name, "", nil, buildPod.Namespace, buildPod.Name).
			WithWait(false).
			WithContainerFilter(selector.FilterTerminatingContainers)
		err = logs.StartLogsWithWriter(ctx, targetselector.NewTargetSelector(options), true, 100, stdoutLogger)
		if err != nil {
			return errors.Errorf("error printing build logs: %v", err)
		}

		ctx.Log().Info("Checking build status...")
		for {
			time.Sleep(time.Second)

			// Check if build was successful
			pod, err := ctx.KubeClient().KubeClient().CoreV1().Pods(b.BuildNamespace).Get(ctx.Context(), buildPodCreated.Name, metav1.GetOptions{})
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
		ctx.Log().Done("Done building image")
		return nil
	}, deleteBuildPod)
	if err != nil {
		// Delete all build pods on error
		labelSelector := fmt.Sprintf("devspace-pid=%s", ctx.RunID())
		pods, getErr := ctx.KubeClient().KubeClient().CoreV1().Pods(b.BuildNamespace).List(ctx.Context(), metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if getErr != nil {
			return err
		}
		for _, pod := range pods.Items {
			_ = ctx.KubeClient().KubeClient().CoreV1().Pods(b.BuildNamespace).Delete(ctx.Context(), pod.Name, metav1.DeleteOptions{})
		}

		return err
	}

	return nil
}
