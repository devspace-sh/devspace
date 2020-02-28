package render

import (
	"io/ioutil"
	"strings"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

func runKubectl(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'kubectl' of test 'render'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "kubectl")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("test 'render' failed: %s %v", f.GetLogContents(), err)
	}

	rc := &cmd.RenderCmd{
		GlobalFlags: &flags.GlobalFlags{},
		SkipPush:    true,
		Tags:        []string{"rM5xKXK"},
	}

	done := utils.Capture()

	err = rc.Run(f, nil, nil)
	if err != nil {
		return err
	}

	capturedOutput, err := done()
	if err != nil {
		return err
	}

	_ = utils.ChangeWorkingDir(f.Pwd+"/tests/render", f.GetLog())
	expectedOutput, err := ioutil.ReadFile("./expectedoutput/kubectl")
	if err != nil {
		return err
	}

	if strings.Index(string(expectedOutput), capturedOutput) == -1 {
		return errors.Errorf("output does not match expected output")
	}

	imagesExpected := 1
	imagesCount := len(f.builtImages)
	if imagesCount != imagesExpected {
		return errors.Errorf("built images expected: %v, found: %v", imagesExpected, imagesCount)
	}

	f.builtImages = map[string]string{}

	return nil
}
