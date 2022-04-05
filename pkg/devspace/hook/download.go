package hook

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
)

func NewDownloadHook() RemoteHook {
	return &remoteDownloadHook{}
}

type remoteDownloadHook struct{}

func (r *remoteDownloadHook) ExecuteRemotely(ctx devspacecontext.Context, hook *latest.HookConfig, podContainer *selector.SelectedPodContainer) error {
	containerPath := "."
	if hook.Download.ContainerPath != "" {
		containerPath = hook.Download.ContainerPath
	}
	localPath := "."
	if hook.Download.LocalPath != "" {
		localPath = hook.Download.LocalPath
	}
	localPath = ctx.ResolvePath(localPath)

	ctx.Log().Infof("Execute hook '%s' in container '%s/%s/%s'", ansi.Color(hookName(hook), "white+b"), podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name)
	ctx.Log().Infof("Copy container '%s' -> local '%s'", containerPath, localPath)
	// Make sure the target folder exists
	destDir := path.Dir(localPath)
	if len(destDir) > 0 {
		_ = os.MkdirAll(destDir, 0755)
	}

	// Download the files
	err := download(ctx.Context(), ctx.KubeClient(), podContainer.Pod, podContainer.Container.Name, localPath, containerPath, ctx.Log())
	if err != nil {
		return errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
	}

	return nil
}

func download(ctx context.Context, client kubectl.Client, pod *k8sv1.Pod, container string, localPath string, containerPath string, log logpkg.Logger) error {
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
		errorChan <- downloadFromPod(ctx, client, pod, container, containerPath, writer)
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

func downloadFromPod(ctx context.Context, client kubectl.Client, pod *k8sv1.Pod, container, containerPath string, writer io.Writer) error {
	stderr := &bytes.Buffer{}
	err := client.ExecStream(ctx, &kubectl.ExecStreamOptions{
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
