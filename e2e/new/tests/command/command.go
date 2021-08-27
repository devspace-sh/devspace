package command

import (
	"bytes"
	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/e2e/new/framework"
	"github.com/onsi/ginkgo"
	"os"
)

var _ = DevSpaceDescribe("command", func() {
	initialDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// create a new factory
	var (
		f *framework.DefaultFactory
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewDefaultFactory()
	})

	ginkgo.It("should run simple command", func() {
		// TODO
	})

	ginkgo.It("should run command with variable", func() {
		// TODO
	})

	ginkgo.It("should run nested command", func() {
		// TODO
	})

	ginkgo.It("should run command from dependency", func() {
		// TODO
	})

	ginkgo.It("should and shouldnt append args", func() {
		tempDir, err := framework.CopyToTempDir("tests/command/testdata/command-appended-args")
		framework.ExpectNoError(err)
		defer framework.CleanupTempDir(initialDir, tempDir)

		stdout := &bytes.Buffer{}
		runCmd := &cmd.RunCmd{
			GlobalFlags: &flags.GlobalFlags{},
			Stdout:      stdout,
			Stderr:      stdout,
		}
		err = runCmd.RunRun(f, nil, []string{"test1", "test123", "test456"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(stdout.String(), "test123 test456")

		stdout = &bytes.Buffer{}
		runCmd = &cmd.RunCmd{
			GlobalFlags: &flags.GlobalFlags{},
			Stdout:      stdout,
			Stderr:      stdout,
		}
		err = runCmd.RunRun(f, nil, []string{"test2", "test123", "test456"})
		framework.ExpectNoError(err)
		framework.ExpectEqual(stdout.String(), "test123 test456")
	})
})
