package kubectl

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"github.com/loft-sh/devspace/pkg/util/log"
	"io"
	"os"
	"path/filepath"

	"github.com/loft-sh/devspace/helper/server/ignoreparser"

	"github.com/loft-sh/devspace/pkg/devspace/sync"

	"github.com/pkg/errors"
	k8sv1 "k8s.io/api/core/v1"
)

// CopyFromReader extracts a tar from the reader to a container path
func (client *client) CopyFromReader(ctx context.Context, pod *k8sv1.Pod, container, containerPath string, reader io.Reader) error {
	_, stderr, err := client.ExecBuffered(ctx, pod, container, []string{"tar", "xzp", "-C", containerPath + "/."}, reader)
	if err != nil {

		if stderr != nil {
			return errors.Errorf("error executing tar: %s: %v", string(stderr), err)
		}

		return errors.Wrap(err, "exec")
	}

	return nil
}

// Copy copies the specified folder to the container
func (client *client) Copy(ctx context.Context, pod *k8sv1.Pod, container, containerPath, localPath string, exclude []string) error {
	// do the actual copy
	reader, writer := io.Pipe()
	errorChan := make(chan error)
	go func() {
		defer reader.Close()
		errorChan <- client.CopyFromReader(ctx, pod, container, containerPath, reader)
	}()
	go func() {
		defer writer.Close()
		errorChan <- writeTar(writer, localPath, exclude)
	}()
	err := <-errorChan
	// wait for the second goroutine to finish
	<-errorChan
	return err
}

func writeTar(writer io.Writer, localPath string, exclude []string) error {
	absolute, err := filepath.Abs(localPath)
	if err != nil {
		return errors.Wrap(err, "absolute")
	}

	// Check if target is there
	stat, err := os.Stat(absolute)
	if err != nil {
		return errors.Wrap(err, "stat")
	}

	// Compile ignore paths
	ignoreMatcher, err := ignoreparser.CompilePaths(exclude, log.Discard)
	if err != nil {
		return errors.Wrap(err, "compile exclude paths")
	}

	// Use compression
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	// When its a file we copy the file to the toplevel of the tar
	if !stat.IsDir() {
		return sync.NewArchiver(filepath.Dir(absolute), tarWriter, ignoreMatcher).AddToArchive(filepath.Base(absolute))
	}

	// When its a folder we copy the contents and not the folder itself to the
	// toplevel of the tar
	return sync.NewArchiver(absolute, tarWriter, ignoreMatcher).AddToArchive("")
}
