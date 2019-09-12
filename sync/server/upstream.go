package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StartUpstreamServer starts a new upstream server with the given reader and writer
func StartUpstreamServer(uploadPath string, reader io.Reader, writer io.Writer, exitOnClose bool) error {
	pipe := util.NewStdStreamJoint(reader, writer, exitOnClose)
	lis := util.NewStdinListener()
	done := make(chan error)

	go func() {
		s := grpc.NewServer()

		remote.RegisterUpstreamServer(s, &Upstream{
			UploadPath: uploadPath,
		})
		reflection.Register(s)

		done <- s.Serve(lis)
	}()

	lis.Ready(pipe)
	return <-done
}

// Upstream is the implementation for the upstream server
type Upstream struct {
	UploadPath string
}

// Remove implements the server
func (u *Upstream) Remove(stream remote.Upstream_RemoveServer) error {
	// Receive file
	for {
		paths, err := stream.Recv()
		if paths != nil {
			for _, path := range paths.Paths {
				// Just remove everything inside and ignore any errors
				os.RemoveAll(filepath.Join(u.UploadPath, path))
			}
		}

		if err == io.EOF {
			return stream.SendAndClose(&remote.Empty{})
		}
		if err != nil {
			return err
		}
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
	defer writer.Close()

	writerErrChan := make(chan error)
	go func() {
		writerErrChan <- u.writeTar(writer, stream)
	}()

	err = untarAll(reader, u.UploadPath, "")
	if err != nil {
		return errors.Wrap(err, "untar all")
	}

	err = <-writerErrChan
	if err != nil {
		return errors.Wrap(err, "write tar")
	}

	return stream.SendAndClose(&remote.Empty{})
}

func (u *Upstream) writeTar(writer io.WriteCloser, stream remote.Upstream_UploadServer) error {
	defer writer.Close()

	// Receive file
	for {
		chunk, err := stream.Recv()
		if chunk != nil {
			n, err := writer.Write(chunk.Content)
			if err != nil {
				return err
			}
			if n != len(chunk.Content) {
				return errors.Errorf("Error writing data: bytes written %d != expected %d", n, len(chunk.Content))
			}
		}

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
