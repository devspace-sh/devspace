package init

import (
	"errors"
	"os"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
)

// CreateDockerfile runs init test with "create docker file" option
func CreateDockerfile(factory *customFactory) error {
	factory.GetLog().Info("Create Dockerfile Test")

	dirPath, dirName, err := utils.CreateTempDir()
	if err != nil {
		return err
	}

	err = utils.ChangeWorkingDir(dirPath)
	if err != nil {
		return err
	}

	// Copy the testdata into the temp dir
	err = utils.Copy(factory.pwd+"/tests/init/testdata/main.go", dirPath+"/main.go")
	if err != nil {
		return err
	}

	defer utils.DeleteTempAndResetWorkingDir(dirPath, factory.pwd)

	port := 8080
	testCase := &initTestCase{
		name:    "Create Dockerfile",
		answers: []string{cmd.CreateDockerfileOption, "go", "Use hub.docker.com => you are logged in as user", "user/" + dirName, "8080"},
		expectedConfig: &latest.Config{
			Version: latest.Version,
			Images: map[string]*latest.ImageConfig{
				"default": &latest.ImageConfig{
					Image: "user/" + dirName,
				},
			},
			Deployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: dirName,
					Helm: &latest.HelmConfig{
						ComponentChart: ptr.Bool(true),
						Values: map[interface{}]interface{}{
							"containers": []interface{}{
								map[interface{}]interface{}{
									"image": "user/" + dirName,
								},
							},
							"service": map[interface{}]interface{}{
								"ports": []interface{}{
									map[interface{}]interface{}{
										"port": port,
									},
								},
							},
						},
					},
				},
			},
			Dev: &latest.DevConfig{
				Ports: []*latest.PortForwardingConfig{
					&latest.PortForwardingConfig{
						ImageName: "default",
						PortMappings: []*latest.PortMapping{
							&latest.PortMapping{
								LocalPort: &port,
							},
						},
					},
				},
				Open: []*latest.OpenConfig{
					&latest.OpenConfig{
						URL: "http://localhost:8080",
					},
				},
				Sync: []*latest.SyncConfig{
					&latest.SyncConfig{
						ImageName:    "default",
						ExcludePaths: []string{"Dockerfile", "devspace.yaml"},
					},
				},
			},
		},
	}

	err = initializeTest(factory, *testCase)
	if err != nil {
		return err
	}

	// Check if Dockerfile has not been created
	if _, err := os.Stat(dirPath + "/Dockerfile"); os.IsNotExist(err) {
		return errors.New("Dockerfile was not created")
	}

	return nil
}
