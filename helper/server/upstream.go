package server

import (
	"context"
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/server/ignoreparser"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/loft-sh/devspace/helper/util/crc32"
	"github.com/loft-sh/devspace/helper/util/pingtimeout"
	"github.com/loft-sh/devspace/helper/util/stderrlog"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/hash"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// UpstreamOptions holds the upstream server options
type UpstreamOptions struct {
	UploadPath  string
	ExludePaths []string

	FileChangeCmd  string
	FileChangeArgs []string

	DirCreateCmd  string
	DirCreateArgs []string

	OverridePermission bool
	ExitOnClose        bool
	Ping               bool
}

// StartUpstreamServer starts a new upstream server with the given reader and writer
func StartUpstreamServer(reader io.Reader, writer io.Writer, options *UpstreamOptions) error {
	pipe := util.NewStdStreamJoint(reader, writer, options.ExitOnClose)
	lis := util.NewStdinListener()
	done := make(chan error)

	// Compile ignore paths
	ignoreMatcher, err := ignoreparser.CompilePaths(options.ExludePaths, logpkg.Discard)
	if err != nil {
		return errors.Wrap(err, "compile paths")
	}

	go func() {
		s := grpc.NewServer()
		upstream := &Upstream{
			options:       options,
			ignoreMatcher: ignoreMatcher,
			ping:          &pingtimeout.PingTimeout{},
		}

		if options.Ping {
			doneChan := make(chan struct{})
			defer close(doneChan)
			upstream.ping.Start(doneChan)
		}

		remote.RegisterUpstreamServer(s, upstream)
		reflection.Register(s)

		done <- s.Serve(lis)
	}()

	lis.Ready(pipe)
	return <-done
}

// Upstream is the implementation for the upstream server
type Upstream struct {
	remote.UnimplementedUpstreamServer

	options *UpstreamOptions

	// ignore matcher is the ignore matcher which matches against excluded files and paths
	ignoreMatcher ignoreparser.IgnoreParser

	ping *pingtimeout.PingTimeout
}

// Ping returns empty
func (u *Upstream) Ping(context.Context, *remote.Empty) (*remote.Empty, error) {
	if u.ping != nil {
		u.ping.Ping()
	}

	return &remote.Empty{}, nil
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
			return stream.SendAndClose(&remote.Empty{})
		}
		if err != nil {
			return err
		}
	}
}

func (u *Upstream) Checksums(ctx context.Context, paths *remote.TouchPaths) (*remote.PathsChecksum, error) {
	if paths != nil {
		// update timestamps & permissions
		stopChan := make(chan struct{})
		go func() {
			for _, path := range paths.Paths {
				if path.Path == "" {
					continue
				}

				// Update timestamp if needed
				absolutePath := filepath.Join(u.options.UploadPath, path.Path)
				if path.MtimeUnix > 0 {
					t := time.Unix(path.MtimeUnix, 0)
					err := os.Chtimes(absolutePath, t, t)
					if err != nil && !os.IsNotExist(err) {
						stderrlog.Infof("Error touching %s: %v", path, err)
					}
				}

				// Update permissions if needed
				if path.Mode > 0 {
					err := os.Chmod(absolutePath, os.FileMode(path.Mode))
					if err != nil && !os.IsNotExist(err) {
						stderrlog.Infof("Error chmod %s: %v", path, err)
					}
				}
			}

			close(stopChan)
		}()

		checksums := make([]uint32, 0, len(paths.Paths))
		for _, path := range paths.Paths {
			if path.Path == "" {
				continue
			}

			// Just remove everything inside and ignore any errors
			absolutePath := filepath.Join(u.options.UploadPath, path.Path)
			checksum, err := crc32.Checksum(absolutePath)
			if err != nil && !os.IsNotExist(err) {
				stderrlog.Infof("Error checksum %s: %v", path, err)
			}

			checksums = append(checksums, checksum)
		}

		<-stopChan
		return &remote.PathsChecksum{Checksums: checksums}, nil
	}

	return &remote.PathsChecksum{Checksums: []uint32{}}, nil
}

func (u *Upstream) removeRecursive(absolutePath string) error {
	files, err := os.ReadDir(absolutePath)
	if err != nil {
		return err
	}

	// Loop over directory contents and check if we should delete the contents
	for _, dirEntry := range files {
		f, err := dirEntry.Info()
		if err != nil {
			continue
		}

		absoluteChildPath := filepath.Join(absolutePath, f.Name())
		if fsutil.IsRecursiveSymlink(f, absoluteChildPath) {
			continue
		}

		// Check if ignored
		if u.ignoreMatcher != nil && !u.ignoreMatcher.RequireFullScan() && u.ignoreMatcher.Matches(absolutePath[len(u.options.UploadPath):], f.IsDir()) {
			continue
		}

		// Remove recursive
		if f.IsDir() {
			// Ignore the errors here
			_ = u.removeRecursive(absoluteChildPath)
		} else {
			// Check if not ignored
			if u.ignoreMatcher == nil || !u.ignoreMatcher.RequireFullScan() || !u.ignoreMatcher.Matches(absolutePath[len(u.options.UploadPath):], false) {
				_ = os.Remove(absoluteChildPath)
			}
		}
	}

	// Check if not ignored
	if u.ignoreMatcher == nil || !u.ignoreMatcher.RequireFullScan() || !u.ignoreMatcher.Matches(absolutePath[len(u.options.UploadPath):], true) {
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
				// this means the tar is done already, so we just exit here
				if strings.Contains(err.Error(), "io: read/write on closed pipe") {
					return nil
				}

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

func (u *Upstream) Execute(ctx context.Context, cmd *remote.Command) (*remote.Empty, error) {
	if cmd.Once {
		hashString := cmd.Cmd
		for _, arg := range cmd.Args {
			hashString += arg
		}

		hashed := hash.String(hashString)
		fileName := "/tmp/devspace-" + hashed
		_, err := os.Stat(fileName)
		if os.IsNotExist(err) {
			err := os.WriteFile(fileName, []byte("1"), 0666)
			if err != nil {
				return nil, errors.Wrap(err, "writing hash file")
			}
		} else if err != nil {
			return nil, errors.Wrap(err, "stat hash file")
		} else {
			return &remote.Empty{}, nil
		}
	}

	out, err := exec.Command(cmd.Cmd, cmd.Args...).CombinedOutput()
	if err != nil {
		return nil, errors.Errorf("Error executing command '%s %s': %s => %v", cmd.Cmd, strings.Join(cmd.Args, " "), string(out), err)
	}

	return &remote.Empty{}, nil
}
