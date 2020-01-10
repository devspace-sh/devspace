package initcmd

import (
	"os"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/pkg/errors"
)

// CreateDockerfile runs init test with "create docker file" option
func CreateDockerfile(f *customFactory, logger log.Logger) error {
	logger.Info("Run sub test 'create_dockerfile' of test 'init'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	err := beforeTest(f, f.GetLog(), "tests/initcmd/testdata/data1")
	defer afterTest(f)
	if err != nil {
		return errors.Errorf("sub test 'create_dockerfile' of 'init' test failed: %s %v", f.GetLogContents(), err)
	}

	port := 8080
	testCase := &initTestCase{
		name:    "Create Dockerfile",
		answers: []string{cmd.CreateDockerfileOption, "go", "Use hub.docker.com => you are logged in as user", "user/" + f.DirName, "8080"},
		expectedConfig: &latest.Config{
			Version: latest.Version,
			Images: map[string]*latest.ImageConfig{
				"default": &latest.ImageConfig{
					Image: "user/" + f.DirName,
				},
			},
			Deployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: f.DirName,
					Helm: &latest.HelmConfig{
						ComponentChart: ptr.Bool(true),
						Values: map[interface{}]interface{}{
							"containers": []interface{}{
								map[interface{}]interface{}{
									"image": "user/" + f.DirName,
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

	err = runTest(f, *testCase)
	if err != nil {
		return err
	}

	// Check if Dockerfile has not been created
	if _, err := os.Stat(f.DirPath + "/Dockerfile"); os.IsNotExist(err) {
		return errors.New("Dockerfile was not created")
	}

	return nil
}
