package server

import (
	"context"
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// UpstreamOptions holds the upstream server options
type UpstreamOptions struct {
	UploadPath  string
	ExludePaths []string

	BatchCmd  string
	BatchArgs []string

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
	ignoreMatcher, err := ignoreparser.CompilePaths(options.ExludePaths)
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
	ignoreMatcher ignoreparser.IgnoreParser
}

// RestartContainer implements the server
func (u *Upstream) RestartContainer(context.Context, *remote.Empty) (*remote.Empty, error) {
	err := util.NewContainerRestarter().RestartContainer()
	if err != nil {
		return nil, err
	}

	return &remote.Empty{}, nil
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
					_ = u.removeRecursive(absolutePath)
				} else {
					_ = os.Remove(absolutePath)
				}
			}
		}

		if err == io.EOF {
			// execute a batch command if needed
			err = u.executeBatchCommand()
			if err != nil {
				return err
			}

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
		if u.ignoreMatcher != nil && u.ignoreMatcher.HasNegatePatterns() == false && util.MatchesPath(u.ignoreMatcher, absolutePath[len(u.options.UploadPath):], f.IsDir()) {
			continue
		}

		// Remove recursive
		if f.IsDir() {
			// Ignore the errors here
			_ = u.removeRecursive(absoluteChildPath)
		} else {
			// Check if not ignored
			if u.ignoreMatcher == nil || u.ignoreMatcher.HasNegatePatterns() == false || util.MatchesPath(u.ignoreMatcher, absolutePath[len(u.options.UploadPath):], false) == false {
				_ = os.Remove(absoluteChildPath)
			}
		}
	}

	// Check if not ignored
	if u.ignoreMatcher == nil || u.ignoreMatcher.HasNegatePatterns() == false || util.MatchesPath(u.ignoreMatcher, absolutePath[len(u.options.UploadPath):], true) == false {
		// This will not remove the directory if there is still a file or directory in it
		return os.Remove(absolutePath)
	}
	return nil
}

// Upload implements the server upload interface and writes all the data received to a
// temporary file
func (u *Upstream) Upload(stream remote.Upstream_UploadServer) error {
	reader, writer := io.Pipe()

	defer reader.Close()
	defer writer.Close()

	writerErrChan := make(chan error)
	go func() {
		writerErrChan <- u.writeTar(writer, stream)
	}()

	err := untarAll(reader, u.options)
	if err != nil {
		return errors.Wrap(err, "untar all")
	}

	err = <-writerErrChan
	if err != nil {
		return errors.Wrap(err, "write tar")
	}

	// execute a batch command if needed
	err = u.executeBatchCommand()
	if err != nil {
		return err
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
			} else if n != len(chunk.Content) {
				return errors.Errorf("error writing data: bytes written %d != expected %d", n, len(chunk.Content))
			}
		}

		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func (u *Upstream) executeBatchCommand() error {
	if u.options.BatchCmd != "" {
		out, err := exec.Command(u.options.BatchCmd, u.options.BatchArgs...).CombinedOutput()
		if err != nil {
			return errors.Errorf("Error executing command '%s %s': %s => %v", u.options.BatchCmd, strings.Join(u.options.BatchArgs, " "), string(out), err)
		}
	}

	return nil
}
