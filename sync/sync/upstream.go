package sync

import (
	"fmt"
	"io"
	"os"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/pkg/errors"
)

// Upstream is the implementation for the upstream server
type Upstream struct {
	UploadPath string
}

// Remove implements the server
func (u *Upstream) Remove(stream remote.Upstream_RemoveServer) error {
	// Receive file
	for {
		path, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&remote.Empty{})
		}
		if err != nil {
			return err
		}

		// Just remove everything inside and ignore any errors
		os.RemoveAll(path.Path)
	}
}

// Upload implements the server upload interface and writes all the data received to a
// temporary file
func (u *Upstream) Upload(stream remote.Upstream_UploadServer) error {
	reader, writer, err := os.Pipe()
	if err != nil {
		return errors.Wrap(err, "pipe")
	}

	defer reader.Close()
	go func() {
		u.writeTar(writer, stream)
	}()

	err = untarAll(reader, u.UploadPath, "")
	if err != nil {
		return errors.Wrap(err, "untar all")
	}

	return stream.SendAndClose(&remote.Empty{})
}

func (u *Upstream) writeTar(writer io.WriteCloser, stream remote.Upstream_UploadServer) error {
	defer writer.Close()

	// Receive file
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		n, err := writer.Write(chunk.Content)
		if err != nil {
			return err
		}
		if n != len(chunk.Content) {
			return fmt.Errorf("Error writing data: bytes written %d != expected %d", n, len(chunk.Content))
		}
	}
}
