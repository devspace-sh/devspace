package render

import (
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/deploy/deployer/helm"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

var chartRegEx = regexp.MustCompile(`component-chart-[^\"]+`)

func replaceComponentChart(in string) string {
	return chartRegEx.ReplaceAllString(in, "component-chart-"+helm.DevSpaceChartConfig.Version)
}

func runHelmV2(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'helm_v2' of test 'render'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, "helm_v2")
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
	expectedOutput, err := ioutil.ReadFile("./expectedoutput/helm_v2")
	if err != nil {
		return err
	}

	expectedOutputStr := replaceComponentChart(string(expectedOutput))
	if strings.Index(capturedOutput, expectedOutputStr) == -1 {
		return errors.Errorf("output '%s' does not match expected output '%s'", capturedOutput, expectedOutputStr)
	}

	imagesExpected := 1
	imagesCount := len(f.builtImages)
	if imagesCount != imagesExpected {
		return errors.Errorf("built images expected: %v, found: %v", imagesExpected, imagesCount)
	}

	f.builtImages = map[string]string{}

	return nil
}
