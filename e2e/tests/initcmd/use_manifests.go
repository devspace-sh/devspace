package initcmd

import (
	"bytes"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// UseManifests runs init test with "use kubernetes manifests" option
func UseManifests(f *customFactory, logger log.Logger) error {
	buff := &bytes.Buffer{}
	f.cacheLogger = NewCustomStreamLogger(buff, logrus.InfoLevel, f.verbose)

	var buffString string
	buffString = buff.String()

	if f.verbose {
		buffString = ""
	}

	logger.Info("Run sub test 'use_manifests' of test 'init'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, f.cacheLogger, "tests/initcmd/testdata/data2")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'use_manifests' of 'init' test failed: %s %v", buffString, err)
	}

	testCase := &initTestCase{
		name:    "Enter kubernetes manifests",
		answers: []string{cmd.EnterManifestsOption, "kube/**"},
		expectedConfig: &latest.Config{
			Version: latest.Version,
			Deployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: f.dirName,
					Kubectl: &latest.KubectlConfig{
						Manifests: []string{
							"kube/**",
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
