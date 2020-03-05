package server

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/sync/remote"
	"github.com/devspace-cloud/devspace/sync/util"
	"github.com/pkg/errors"
	gitignore "github.com/sabhiram/go-gitignore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// UpstreamOptions holds the upstream server options
type UpstreamOptions struct {
	UploadPath  string
	ExludePaths []string

	FileChangeCmd  string
	FileChangeArgs []string

	DirCreateCmd  string
	DirCreateArgs []string

	ExitOnClose bool
}

// StartUpstreamServer starts a new upstream server with the given reader and writer
func StartUpstreamServer(reader io.Reader, writer io.Writer, options *UpstreamOptions) error {
	pipe := util.NewStdStreamJoint(reader, writer, options.ExitOnClose)
	lis := util.NewStdinListener()
	done := make(chan error)

	// Compile ignore paths
	ignoreMatcher, err := compilePaths(options.ExludePaths)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	go func() {
		s := grpc.NewServer()

		remote.RegisterUpstreamServer(s, &Upstream{
			options:       options,
			ignoreMatcher: ignoreMatcher,
		})
		reflection.Register(s)

		done <- s.Serve(lis)
	}()

	lis.Ready(pipe)
	return <-done
}

// Upstream is the implementation for the upstream server
type Upstream struct {
	options *UpstreamOptions

	// ignore matcher is the ignore matcher which matches against excluded files and paths
	ignoreMatcher gitignore.IgnoreParser
}

// Remove implements the server
func (u *Upstream) Remove(stream remote.Upstream_RemoveServer) error {
	// Receive file
	for {
		paths, err := stream.Recv()
		if paths != nil {
			for _, path := range paths.Paths {
				// Just remove everything inside and ignore any errors
				absolutePath := filepath.Join(u.options.UploadPath, path)

				// Stat the path
				stat, err := os.Stat(absolutePath)
				if err != nil {
					continue
				}

				if stat.IsDir() {
					u.removeRecursive(absolutePath)
				} else {
					os.Remove(absolutePath)
				}
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

func (u *Upstream) removeRecursive(absolutePath string) error {
	files, err := ioutil.ReadDir(absolutePath)
	if err != nil {
		return err
	}

	// Loop over directory contents and check if we should delete the contents
	for _, f := range files {
		absoluteChildPath := filepath.Join(absolutePath, f.Name())

		// Check if ignored
		if u.ignoreMatcher != nil && util.MatchesPath(u.ignoreMatcher, absolutePath[len(u.options.UploadPath):], f.IsDir()) {
			continue
		}

		// Remove recursive
		if f.IsDir() {
			// Ignore the errors here
			_ = u.removeRecursive(absoluteChildPath)
		} else {
			os.Remove(absoluteChildPath)
		}
	}

	// This will not remove the directory if there is still a file or directory in it
	return os.Remove(absolutePath)
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

	err = untarAll(reader, u.options)
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
