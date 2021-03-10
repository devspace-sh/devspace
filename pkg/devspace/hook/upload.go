package hook

import (
	"archive/tar"
	"compress/gzip"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"os"
	"path"
	"path/filepath"
)

func NewUploadHook() RemoteHook {
	return &remoteUploadHook{}
}

type remoteUploadHook struct{}

func (r *remoteUploadHook) ExecuteRemotely(ctx Context, hook *latest.HookConfig, podContainer *kubectl.SelectedPodContainer, log logpkg.Logger) error {
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
			return errors.Errorf("error in container '%s/%s/%s': %v: %s", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err, string(stderr))
		}
	}

	// Upload the files
	err := upload(ctx.Client, podContainer.Pod, podContainer.Container.Name, localPath, containerPath)
	if err != nil {
		return errors.Errorf("error in container '%s/%s/%s': %v", podContainer.Pod.Namespace, podContainer.Pod.Name, podContainer.Container.Name, err)
	}

	return nil
}

func upload(client kubectl.Client, pod *v1.Pod, container string, localPath string, containerPath string) error {
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

func uploadFromReader(client kubectl.Client, pod *v1.Pod, container, containerPath string, reader io.Reader) error {
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
