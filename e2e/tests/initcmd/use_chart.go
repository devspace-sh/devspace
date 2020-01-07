package initcmd

import (
	"bytes"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// UseChart runs init test with "use helm chart" option
func UseChart(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = NewCustomStreamLogger(buff, logrus.InfoLevel)

	logger.Info("Run sub test 'use_chart' of test 'init'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	client, err := f.NewKubeClientFromContext("", f.namespace, false)
	if err != nil {
		return errors.Errorf("Unable to create new kubectl client: %v", err)
	}

	// The client is saved in the f ONCE for each sub test
	f.client = client

	err = beforeTest(f, f.cacheLogger, "tests/initcmd/testdata/data2")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'use_chart' of 'init' test failed: %s %v", buff.String(), err)
	}

	testCase := &initTestCase{
		name:    "Enter helm chart",
		answers: []string{cmd.EnterHelmChartOption, "./chart"},
		expectedConfig: &latest.Config{
			Version: latest.Version,
			Deployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: f.dirName,
					Helm: &latest.HelmConfig{
						Chart: &latest.ChartConfig{
							Name: "./chart",
						},
					},
				},
			},
			Dev:    &latest.DevConfig{},
			Images: latest.NewRaw().Images,
		},
	}

	err = runTest(f, *testCase)
	if err != nil {
		return err
	}

	return nil
}
