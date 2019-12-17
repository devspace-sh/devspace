package init

import (
	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
)

// UseManifests runs init test with "use kubernetes manifests" option
func UseManifests(factory *customFactory) error {
	dirPath, dirName, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	err = utils.Copy(factory.pwd+"/tests/init/testdata", dirPath)
	if err != nil {
		return err
	}

	err = utils.ChangeWorkingDir(dirPath)
	if err != nil {
		return err
	}

	defer utils.DeleteTempAndResetWorkingDir(dirPath, factory.pwd)

	testCase := &initTestCase{
		name:    "Enter kubernetes manifests",
		answers: []string{cmd.EnterManifestsOption, "kube/**"},
		expectedConfig: &latest.Config{
			Version: latest.Version,
			Deployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: dirName,
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

	err = initializeTest(factory, *testCase)
	if err != nil {
		return err
	}

	return nil
}
