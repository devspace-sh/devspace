package hook

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/services/targetselector"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/command"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	dockerterm "github.com/docker/docker/pkg/term"
)

const (
	KubeContextEnv   = "DEVSPACE_HOOK_KUBE_CONTEXT"
	KubeNamespaceEnv = "DEVSPACE_HOOK_KUBE_NAMESPACE"
	ErrorEnv         = "DEVSPACE_HOOK_ERROR"
	OsArgsEnv        = "DEVSPACE_HOOK_OS_ARGS"
)

// Executer executes configured commands locally
type Executer interface {
	OnError(stage Stage, whichs []string, context Context, log logpkg.Logger)
	Execute(when When, stage Stage, which string, context Context, log logpkg.Logger) error
	ExecuteMultiple(when When, stage Stage, whichs []string, context Context, log logpkg.Logger) error
}

type executer struct {
	config *latest.Config
}

// NewExecuter creates an instance of Executer for the specified config
func NewExecuter(config *latest.Config) Executer {
	return &executer{
		config: config,
	}
}

// When is the type that is used to tell devspace when relatively to a stage a hook should be executed
type When string

const (
	// Before is used to tell devspace to execute a hook before a certain stage
	Before When = "before"
	// After is used to tell devspace to execute a hook after a certain stage
	After When = "after"
	// OnError is used to tell devspace to execute a hook after a certain error occured
	OnError When = "onError"
)

// Stage is the type that defines the stage at when to execute a hook
type Stage string

const (
	// StageImages is the image building stage
	StageImages Stage = "images"
	// StageDeployments is the deploying stage
	StageDeployments Stage = "deployments"
	// StagePurgeDeployments is the purging stage
	StagePurgeDeployments Stage = "purgeDeployments"
	// StageDependencies is the dependency stage
	StageDependencies Stage = "dependencies"
	// StagePullSecrets is the pull secrets stage
	StagePullSecrets Stage = "pullSecrets"
)

// All is used to tell devspace to execute a hook before or after all images, deployments
const All = "all"

var (
	_, stdout, stderr = dockerterm.StdStreams()
)

// Context holds hook context information
type Context struct {
	Error  error
	Client kubectl.Client
	Config *latest.Config
	Cache  *generated.CacheConfig
}

// ExecuteMultiple executes multiple hooks at a specific time
func (e *executer) ExecuteMultiple(when When, stage Stage, whichs []string, context Context, log logpkg.Logger) error {
	for _, which := range whichs {
		err := e.Execute(when, stage, which, context, log)
		if err != nil {
			return err
		}
	}

	return nil
}

// OnError is a convience method to handle the resulting error of a hook execution. Since we mostly return anyways after
// an error has occured this only prints additonal information why the hook failed
func (e *executer) OnError(stage Stage, whichs []string, context Context, log logpkg.Logger) {
	err := e.ExecuteMultiple(OnError, stage, whichs, context, log)
	if err != nil {
		log.Warnf("Hook failed: %v", err)
	}
}

// Execute executes hooks at a specific time
func (e *executer) Execute(when When, stage Stage, which string, context Context, log logpkg.Logger) error {
	if e.config.Hooks != nil && len(e.config.Hooks) > 0 {
		hooksToExecute := []*latest.HookConfig{}

		// Gather all hooks we should execute
		for _, hook := range e.config.Hooks {
			if hook.When != nil {
				if when == Before && hook.When.Before != nil {
					if stage == StageDeployments && hook.When.Before.Deployments != "" && strings.TrimSpace(hook.When.Before.Deployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.Before.PurgeDeployments != "" && strings.TrimSpace(hook.When.Before.PurgeDeployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.Before.Images != "" && strings.TrimSpace(hook.When.Before.Images) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.Before.Dependencies != "" && strings.TrimSpace(hook.When.Before.Dependencies) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.Before.PullSecrets != "" && strings.TrimSpace(hook.When.Before.PullSecrets) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				} else if when == After && hook.When.After != nil {
					if stage == StageDeployments && hook.When.After.Deployments != "" && strings.TrimSpace(hook.When.After.Deployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.After.PurgeDeployments != "" && strings.TrimSpace(hook.When.Before.PurgeDeployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.After.Images != "" && strings.TrimSpace(hook.When.After.Images) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.After.Dependencies != "" && strings.TrimSpace(hook.When.After.Dependencies) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.After.PullSecrets != "" && strings.TrimSpace(hook.When.After.PullSecrets) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				} else if when == OnError && hook.When.OnError != nil {
					if stage == StageDeployments && hook.When.OnError.Deployments != "" && strings.TrimSpace(hook.When.OnError.Deployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePurgeDeployments && hook.When.OnError.PurgeDeployments != "" && strings.TrimSpace(hook.When.Before.PurgeDeployments) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageImages && hook.When.OnError.Images != "" && strings.TrimSpace(hook.When.OnError.Images) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StageDependencies && hook.When.OnError.Dependencies != "" && strings.TrimSpace(hook.When.OnError.Dependencies) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					} else if stage == StagePullSecrets && hook.When.OnError.PullSecrets != "" && strings.TrimSpace(hook.When.OnError.PullSecrets) == strings.TrimSpace(which) {
						hooksToExecute = append(hooksToExecute, hook)
					}
				}
			}
		}

		// Execute hooks
		for _, hook := range hooksToExecute {
			if command.ShouldExecuteOnOS(hook.OperatingSystem) == false {
				continue
			}

			// Determine output writer
			var writer io.Writer
			if log == logpkg.GetInstance() {
				writer = stdout
			} else {
				writer = log
			}

			// Where to execute
			execute := executeLocally
			if hook.Where.Container != nil {
				execute = executeInContainer
			}

			// Execute the hook
			err := executeHook(context, hook, writer, log, execute)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func executeHook(ctx Context, hook *latest.HookConfig, writer io.Writer, log logpkg.Logger, execute func(Context, *latest.HookConfig, io.Writer, logpkg.Logger) error) error {
	var (
		hookLog    logpkg.Logger
		hookWriter io.Writer
	)
	if hook.Silent {
		hookLog = logpkg.Discard
		hookWriter = &bytes.Buffer{}
	} else {
		hookLog = log
		hookWriter = writer
	}

	if hook.Background {
		log.Infof("Execute hook '%s' in background", ansi.Color(hookName(hook), "white+b"))
		go func() {
			err := execute(ctx, hook, hookWriter, hookLog)
			if err != nil {
				if hook.Silent {
					log.Warnf("Error executing hook '%s' in background: %s %v", ansi.Color(hookName(hook), "white+b"), hookWriter.(*bytes.Buffer).String(), err)
				} else {
					log.Warnf("Error executing hook '%s' in background: %v", ansi.Color(hookName(hook), "white+b"), err)
				}
			}
		}()

		return nil
	}

	log.Infof("Execute hook '%s'", ansi.Color(hookName(hook), "white+b"))
	err := execute(ctx, hook, hookWriter, hookLog)
	if err != nil {
		if hook.Silent {
			return errors.Wrapf(err, "in hook '%s': %s", ansi.Color(hookName(hook), "white+b"), hookWriter.(*bytes.Buffer).String())
		} else {
			return errors.Wrapf(err, "in hook '%s'", ansi.Color(hookName(hook), "white+b"))
		}
	}

	return nil
}

func executeLocally(ctx Context, hook *latest.HookConfig, writer io.Writer, log logpkg.Logger) error {
	// Create extra env variables
	osArgsBytes, err := json.Marshal(os.Args)
	if err != nil {
		return err
	}
	extraEnv := map[string]string{
		OsArgsEnv: string(osArgsBytes),
	}
	if ctx.Client != nil {
		extraEnv[KubeContextEnv] = ctx.Client.CurrentContext()
		extraEnv[KubeNamespaceEnv] = ctx.Client.Namespace()
	}
	if ctx.Error != nil {
		extraEnv[ErrorEnv] = ctx.Error.Error()
	}

	err = command.ExecuteCommandWithEnv(hook.Command, hook.Args, writer, writer, extraEnv)
	if err != nil {
		return err
	}

	return nil
}

func executeInContainer(ctx Context, hook *latest.HookConfig, writer io.Writer, log logpkg.Logger) error {
	if ctx.Client == nil {
		return errors.Errorf("Cannot execute hook '%s': kube client is not initialized", ansi.Color(hookName(hook), "white+b"))
	}

	var imageSelector []string
	if hook.Where.Container.ImageName != "" {
		if ctx.Config == nil || ctx.Cache == nil {
			return errors.Errorf("Cannot execute hook '%s': config is not loaded", ansi.Color(hookName(hook), "white+b"))
		}

		imageSelector = targetselector.ImageSelectorFromConfig(hook.Where.Container.ImageName, ctx.Config, ctx.Cache)
	}

	if hook.Where.Container.Wait == nil || *hook.Where.Container.Wait == true {
		log.Infof("Waiting for running containers for hook '%s'", ansi.Color(hookName(hook), "white+b"))

		timeout := time.Second * 120
		if hook.Where.Container.Timeout > 0 {
			timeout = time.Duration(hook.Where.Container.Timeout) * time.Second
		}

		err := wait.Poll(time.Second, timeout, func() (done bool, err error) {
			return executeInFoundContainer(ctx, hook, imageSelector, writer, log)
		})
		if err != nil {
			if err == wait.ErrWaitTimeout {
				return errors.Errorf("timeout: couldn't find a running container")
			}

			return err
		}

		return nil
	}

	executed, err := executeInFoundContainer(ctx, hook, imageSelector, writer, log)
	if err != nil {
		return err
	} else if executed == false {
		log.Infof("Skip hook '%s', because no running containers were found", ansi.Color(hookName(hook), "white+b"))
	}
	return nil
}

func executeInFoundContainer(ctx Context, hook *latest.HookConfig, imageSelector []string, writer io.Writer, log logpkg.Logger) (bool, error) {
	labelSelector := ""
	if len(hook.Where.Container.LabelSelector) > 0 {
		labelSelector = labels.Set(hook.Where.Container.LabelSelector).String()
	}

	podContainers, err := kubectl.NewFilterWithSort(ctx.Client, kubectl.SortPodsByNewest, kubectl.SortContainersByNewest).SelectContainers(context.TODO(), kubectl.Selector{
		ImageSelector: imageSelector,
		LabelSelector: labelSelector,
		Pod:           hook.Where.Container.Pod,
		ContainerName: hook.Where.Container.ContainerName,
		Namespace:     hook.Where.Container.Namespace,
	})
	if err != nil {
		return false, err
	} else if len(podContainers) == 0 {
		return false, nil
	}

	// if any podContainer is not running we wait
	for _, podContainer := range podContainers {
		if targetselector.IsContainerRunning(podContainer) == false {
			return false, nil
		}
	}

	// execute the hook in the containers
	for _, podContainer := range podContainers {
		log.Infof("Execute hook '%s' in container '%s/%s/%s'", ansi.Color(hookName(hook), "white+b"), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name)
		if hook.Download != nil {
			containerPath := "."
			if hook.Download.ContainerPath != "" {
				containerPath = hook.Download.ContainerPath
			}
			localPath := "."
			if hook.Download.LocalPath != "" {
				localPath = hook.Download.LocalPath
			}

			log.Infof("Copy container '%s' -> local '%s'", containerPath, localPath)
			// Make sure the target folder exists
			destDir := path.Dir(localPath)
			if len(destDir) > 0 {
				_ = os.MkdirAll(destDir, 0755)
			}

			// Download the files
			err = download(ctx.Client, podContainer.Pod, podContainer.Container.Name, localPath, containerPath, log)
			if err != nil {
				return false, errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
			}
		} else if hook.Upload != nil {
			containerPath := "."
			if hook.Upload.ContainerPath != "" {
				containerPath = hook.Upload.ContainerPath
			}
			localPath := "."
			if hook.Upload.LocalPath != "" {
				localPath = hook.Upload.LocalPath
			}

			log.Infof("Copy local '%s' -> container '%s'", localPath, containerPath)
			// Make sure the target folder exists
			destDir := path.Dir(containerPath)
			if len(destDir) > 0 {
				_, stderr, err := ctx.Client.ExecBuffered(podContainer.Pod, podContainer.Container.Name, []string{"mkdir", "-p", destDir}, nil)
				if err != nil {
					return false, errors.Errorf("error in container '%s/%s/%s': %v: %s", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err, string(stderr))
				}
			}

			// Upload the files
			err = upload(ctx.Client, podContainer.Pod, podContainer.Container.Name, localPath, containerPath)
			if err != nil {
				return false, errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
			}
		} else {
			cmd := []string{hook.Command}
			cmd = append(cmd, hook.Args...)
			err = ctx.Client.ExecStream(&kubectl.ExecStreamOptions{
				Pod:       podContainer.Pod,
				Container: podContainer.Container.Name,
				Command:   cmd,
				Stdout:    writer,
				Stderr:    writer,
			})
			if err != nil {
				return false, errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
			}
		}
	}

	return true, nil
}

func hookName(hook *latest.HookConfig) string {
	if hook.Command != "" {
		return fmt.Sprintf("%s %s", hook.Command, strings.Join(hook.Args, " "))
	}
	if hook.Upload != nil && hook.Where.Container != nil {
		localPath := "."
		if hook.Upload.LocalPath != "" {
			localPath = hook.Upload.LocalPath
		}
		containerPath := "."
		if hook.Upload.ContainerPath != "" {
			containerPath = hook.Upload.ContainerPath
		}

		if hook.Where.Container.Pod != "" {
			return fmt.Sprintf("copy %s to pod %s", localPath, hook.Where.Container.Pod)
		}
		if len(hook.Where.Container.LabelSelector) > 0 {
			return fmt.Sprintf("copy %s to selector %s", localPath, labels.Set(hook.Where.Container.LabelSelector).String())
		}
		if hook.Where.Container.ImageName != "" {
			return fmt.Sprintf("copy %s to imageName %s", localPath, hook.Where.Container.ImageName)
		}

		return fmt.Sprintf("copy %s to %s", localPath, containerPath)
	}
	return "hook"
}

func upload(client kubectl.Client, pod *k8sv1.Pod, container string, localPath string, containerPath string) error {
	// do the actual copy
	reader, writer := io.Pipe()
	errorChan := make(chan error)
	go func() {
		defer reader.Close()
		errorChan <- uploadFromReader(client, pod, container, containerPath, reader)
	}()
	go func() {
		defer writer.Close()
		errorChan <- makeTar(localPath, containerPath, writer)
	}()
	err := <-errorChan
	// wait for the second goroutine to finish
	<-errorChan
	return err
}

func uploadFromReader(client kubectl.Client, pod *k8sv1.Pod, container, containerPath string, reader io.Reader) error {
	cmd := []string{"tar", "xzp"}
	destDir := path.Dir(containerPath)
	if len(destDir) > 0 {
		cmd = append(cmd, "-C", destDir)
	}

	_, stderr, err := client.ExecBuffered(pod, container, cmd, reader)
	if err != nil {
		if stderr != nil {
			return errors.Errorf("error executing tar: %s: %v", string(stderr), err)
		}

		return errors.Wrap(err, "exec")
	}

	return nil
}

func makeTar(srcPath, destPath string, writer io.Writer) error {
	gw := gzip.NewWriter(writer)
	defer gw.Close()
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)
	return recursiveTar(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter)
}

func recursiveTar(srcBase, srcFile, destBase, destFile string, tw *tar.Writer) error {
	srcPath := path.Join(srcBase, srcFile)
	matchedPaths, err := filepath.Glob(srcPath)
	if err != nil {
		return err
	}
	for _, fpath := range matchedPaths {
		stat, err := os.Lstat(fpath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			files, err := ioutil.ReadDir(fpath)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				//case empty directory
				hdr, _ := tar.FileInfoHeader(stat, fpath)
				hdr.Name = destFile
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
			}
			for _, f := range files {
				if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw); err != nil {
					return err
				}
			}
			return nil
		} else if stat.Mode()&os.ModeSymlink != 0 {
			//case soft link
			hdr, _ := tar.FileInfoHeader(stat, fpath)
			target, err := os.Readlink(fpath)
			if err != nil {
				return err
			}

			hdr.Linkname = target
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			//case regular file or other file type like pipe
			hdr, err := tar.FileInfoHeader(stat, fpath)
			if err != nil {
				return err
			}
			hdr.Name = destFile

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			f, err := os.Open(fpath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			return f.Close()
		}
	}
	return nil
}

func download(client kubectl.Client, pod *k8sv1.Pod, container string, localPath string, containerPath string, log logpkg.Logger) error {
	prefix := getPrefix(containerPath)
	prefix = path.Clean(prefix)
	// remove extraneous path shortcuts - these could occur if a path contained extra "../"
	// and attempted to navigate beyond "/" in a remote filesystem
	prefix = stripPathShortcuts(prefix)

	// do the actual copy
	reader, writer := io.Pipe()
	errorChan := make(chan error)
	go func() {
		defer writer.Close()
		errorChan <- downloadFromPod(client, pod, container, containerPath, writer)
	}()
	go func() {
		defer reader.Close()
		errorChan <- untarAll(reader, localPath, prefix, log)
	}()
	err := <-errorChan
	// wait for the second goroutine to finish
	<-errorChan
	return err
}

func downloadFromPod(client kubectl.Client, pod *k8sv1.Pod, container, containerPath string, writer io.Writer) error {
	stderr := &bytes.Buffer{}
	err := client.ExecStream(&kubectl.ExecStreamOptions{
		Pod:       pod,
		Container: container,
		Command:   []string{"tar", "czf", "-", containerPath},
		Stdout:    writer,
		Stderr:    stderr,
	})
	if err != nil {
		return errors.Errorf("error executing tar: %s: %v", stderr.String(), err)
	}

	return nil
}

func getPrefix(file string) string {
	// tar strips the leading '/' if it's there, so we will too
	return strings.TrimLeft(file, "/")
}

func untarAll(reader io.Reader, destDir, prefix string, log logpkg.Logger) error {
	gw, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	symlinkWarningPrinted := false
	tarReader := tar.NewReader(gw)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		// All the files will start with the prefix, which is the directory where
		// they were located on the pod, we need to strip down that prefix, but
		// if the prefix is missing it means the tar was tempered with.
		// For the case where prefix is empty we need to ensure that the path
		// is not absolute, which also indicates the tar file was tempered with.
		if !strings.HasPrefix(header.Name, prefix) {
			return fmt.Errorf("tar contents corrupted")
		}

		// basic file information
		mode := header.FileInfo().Mode()
		destFileName := filepath.Join(destDir, header.Name[len(prefix):])

		if !isDestRelative(destDir, destFileName) {
			log.Warnf("warning: file %q is outside target destination, skipping", destFileName)
			continue
		}

		baseName := filepath.Dir(destFileName)
		if err := os.MkdirAll(baseName, 0755); err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(destFileName, 0755); err != nil {
				return err
			}
			continue
		}

		if mode&os.ModeSymlink != 0 {
			if !symlinkWarningPrinted {
				symlinkWarningPrinted = true
				log.Warnf("warning: skipping symlink: %q -> %q\n", destFileName, header.Linkname)
			}
			continue
		}
		outFile, err := os.Create(destFileName)
		if err != nil {
			return err
		}
		defer outFile.Close()
		if _, err := io.Copy(outFile, tarReader); err != nil {
			return err
		}
		if err := outFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

// isDestRelative returns true if dest is pointing outside the base directory,
// false otherwise.
func isDestRelative(base, dest string) bool {
	relative, err := filepath.Rel(base, dest)
	if err != nil {
		return false
	}
	return relative == "." || relative == stripPathShortcuts(relative)
}

// stripPathShortcuts removes any leading or trailing "../" from a given path
func stripPathShortcuts(p string) string {
	newPath := path.Clean(p)
	trimmed := strings.TrimPrefix(newPath, "../")

	for trimmed != newPath {
		newPath = trimmed
		trimmed = strings.TrimPrefix(newPath, "../")
	}

	// trim leftover {".", ".."}
	if newPath == "." || newPath == ".." {
		newPath = ""
	}

	if len(newPath) > 0 && string(newPath[0]) == "/" {
		return newPath[1:]
	}

	return newPath
}
