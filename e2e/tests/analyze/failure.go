package analyze

import (
	"strings"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/pkg/errors"

	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
)

func runFailure(f *utils.BaseCustomFactory, logger log.Logger) error {
	logger.Info("Run sub test 'failure' of test 'analyze'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	f.DirName = "failure"

	ac := &cmd.AnalyzeCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
		},
		Wait: true,
	}

	err := beforeTest(f)
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'failure' of 'analyze' test failed: %s %v", f.GetLogContents(), err)
	}

	verboseHistory := f.Verbose

	f.Verbose = false

	err = ac.RunAnalyze(f, nil, nil)
	if err != nil {
		return errors.Errorf("err should be nil: %v", err)
	}

	expectedOutput := []string{"ErrImagePull", "pull access denied for randomimage123", "test-fail"}
	output := []string{}

	for _, eo := range expectedOutput {
		if strings.Index(f.GetLogContents(), eo) != -1 && !utils.StringInSlice(eo, output) {
			// Success, pod error found
			output = append(output, eo)
		}
	}

	f.Verbose = verboseHistory

	if len(output) != len(expectedOutput) {
		return errors.Errorf("expectedOutput '%+v' is different than output '%+v'", expectedOutput, output)
	}

	return nil
}
