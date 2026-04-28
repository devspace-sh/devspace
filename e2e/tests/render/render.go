package render

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/onsi/ginkgo/v2"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/framework"
	"github.com/loft-sh/devspace/pkg/util/factory"
)

var _ = DevSpaceDescribe("build", func() {

	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var f factory.Factory

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()
	})

	// Test cases:

	ginkgo.It("should render helm charts", func() {
		tempDir, err := framework.CopyToTempDir("tests/render/testdata/helm")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		stdout := &Buffer{}
		// create build command
		renderCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			Pipeline:     "deploy",
			Render:       true,
			RenderWriter: stdout,
			SkipPush:     true,
		}
		err = renderCmd.RunDefault(f)
		framework.ExpectNoError(err)
		content := strings.TrimSpace(stdout.String()) + "\n"

		framework.ExpectLocalFileContentsImmediately(
			filepath.Join(tempDir, "rendered.txt"),
			content,
		)
	})

	ginkgo.It("should render kubectl deployments", func() {
		tempDir, err := framework.CopyToTempDir("tests/render/testdata/kubectl")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		stdout := &Buffer{}
		// create build command
		renderCmd := &cmd.RunPipelineCmd{
			GlobalFlags: &flags.GlobalFlags{
				NoWarn: true,
			},
			SkipPush:     true,
			Pipeline:     "deploy",
			Render:       true,
			RenderWriter: stdout,
		}
		err = renderCmd.RunDefault(f)
		framework.ExpectNoError(err)
		content := strings.TrimSpace(stdout.String()) + "\n"
		framework.ExpectLocalFileContentsImmediately(
			filepath.Join(tempDir, "rendered.txt"),
			content,
		)
	})
})

// Buffer is a goroutine safe bytes.Buffer
type Buffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (s *Buffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the Buffer is a nil pointer, it returns "<nil>".
func (s *Buffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}
