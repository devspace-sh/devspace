package add

/*
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/config"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/v1beta3"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type addDeploymentTestCase struct {
	name string

	fakeConfig config.Config

	args    []string
	answers []string
	cmd     *deploymentCmd

	expectedErr    string
	expectedConfig latest.Config
}

func TestRunAddDeployment(t *testing.T) {
	testCases := []addDeploymentTestCase{
		addDeploymentTestCase{
			name: "Add already existing deployment",
			args: []string{"exists"},
			fakeConfig: &latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "exists",
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{""},
						},
					},
				},
			},
			expectedErr: "Deployment exists already exists",
			expectedConfig: latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "exists",
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{""},
						},
					},
				},
				Dev: &latest.DevConfig{},
			},
		},
		addDeploymentTestCase{
			name: "Valid kubectl deployment",
			args: []string{"newKubectlDeployment"},
			fakeConfig: &latest.Config{
				Version: "v1beta3",
			},

			cmd: &deploymentCmd{
				Manifests: "these, are, manifests",
				GlobalFlags: &flags.GlobalFlags{
					Namespace: "kubectlNamespace",
				},
			},

			expectedConfig: latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name:      "newKubectlDeployment",
						Namespace: "kubectlNamespace",
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{"these", "are", "manifests"},
						},
					},
				},
				Dev: &latest.DevConfig{},
			},
		},
		addDeploymentTestCase{
			name: "Valid helm deployment",
			args: []string{"newHelmDeployment"},
			fakeConfig: &latest.Config{
				Version: "v1beta3",
			},

			cmd: &deploymentCmd{
				Chart:        "myChart",
				ChartRepo:    "myChartRepo",
				ChartVersion: "myChartVersion",
			},

			expectedConfig: latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "newHelmDeployment",
						Helm: &latest.HelmConfig{
							Chart: &latest.ChartConfig{
								Name:    "myChart",
								RepoURL: "myChartRepo",
								Version: "myChartVersion",
							},
						},
					},
				},
				Dev: &latest.DevConfig{},
			},
		},
		addDeploymentTestCase{
			name:    "Valid dockerfile deployment",
			args:    []string{"newDockerfileDeployment"},
			answers: []string{"1234"},
			fakeConfig: &latest.Config{
				Version: "v1beta3",
			},

			cmd: &deploymentCmd{
				Dockerfile: "myDockerfile",
				Image:      "myImage",
				Context:    "myContext",
			},

			expectedConfig: latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "newDockerfileDeployment",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []latest.ImageConfig{
									latest.ImageConfig{
										Image: "myImage",
									},
								},
								"service": latest.ServiceConfig{
									Ports: []*latest.ServicePortConfig{
										&latest.ServicePortConfig{
											Port: ptr.Int(1234),
										},
									},
								},
							},
						},
					},
				},
				Images: map[string]*latest.ImageConfig{
					"newDockerfileDeployment": &latest.ImageConfig{
						Image:            "myImage",
						Dockerfile:       "myDockerfile",
						Context:          "myContext",
						CreatePullSecret: ptr.Bool(true),
					},
				},
				Dev: &latest.DevConfig{},
			},
		},
		addDeploymentTestCase{
			name:    "Valid image deployment",
			args:    []string{"newImageDeployment"},
			answers: []string{"1234"},
			fakeConfig: &v1beta3.Config{
				Version: "v1beta3",
				Images: map[string]*v1beta3.ImageConfig{
					"someImage": &v1beta3.ImageConfig{
						Image: "someImage",
					},
				},
			},

			cmd: &deploymentCmd{
				Image:   "myImage",
				Context: "myContext",
			},

			expectedConfig: latest.Config{
				Version: latest.Version,
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "newImageDeployment",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []latest.ImageConfig{
									latest.ImageConfig{
										Image: "myImage",
									},
								},
								"service": latest.ServiceConfig{
									Ports: []*latest.ServicePortConfig{
										&latest.ServicePortConfig{
											Port: ptr.Int(1234),
										},
									},
								},
							},
						},
					},
				},
				Images: map[string]*latest.ImageConfig{
					"someImage": &latest.ImageConfig{
						Image: "someImage",
					},
					"newImageDeployment": &latest.ImageConfig{
						Image:            "myImage",
						Tag:              "latest",
						CreatePullSecret: ptr.Bool(true),
						Build: &latest.BuildConfig{
							Disabled: ptr.Bool(true),
						},
					},
				},
				Dev: &latest.DevConfig{},
			},
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testRunAddDeployment(t, testCase)
	}
}

func testRunAddDeployment(t *testing.T, testCase addDeploymentTestCase) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	if testCase.fakeConfig != nil {
		fakeConfigAsYaml, err := yaml.Marshal(testCase.fakeConfig)
		assert.NilError(t, err, "Error parsing fakeConfig into yaml in testCase %s", testCase.name)
		fsutil.WriteToFile(fakeConfigAsYaml, constants.DefaultConfigPath)
	}
	loader.ResetConfig()

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	if testCase.cmd == nil {
		testCase.cmd = &deploymentCmd{}
	}
	if testCase.cmd.GlobalFlags == nil {
		testCase.cmd.GlobalFlags = &flags.GlobalFlags{}
	}

	err = (testCase.cmd).RunAddDeployment(nil, testCase.args)
	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}

	config, err := loadConfigFromPath()
	if err != nil {
		configContent, _ := fsutil.ReadFile(constants.DefaultConfigPath, -1)
		t.Fatalf("Error loading config after adding deployment in testCase %s: \n%v\n Content of %s:\n%s", testCase.name, err, constants.DefaultConfigPath, string(configContent))
	}

	configAsYaml, _ := yaml.Marshal(config)
	expectedAsYaml, _ := yaml.Marshal(testCase.expectedConfig)
	assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected config in testCase %s.\nExpected:\n%s\nActual config:\n%s", testCase.name, string(expectedAsYaml), string(configAsYaml))
}

func loadConfigFromPath() (*latest.Config, error) {
	yamlFileContent, err := ioutil.ReadFile(constants.DefaultConfigPath)
	if err != nil {
		return nil, err
	}

	oldConfig := map[interface{}]interface{}{}
	err = yaml.Unmarshal(yamlFileContent, oldConfig)
	if err != nil {
		return nil, err
	}

	newConfig, err := versions.Parse(oldConfig, nil, log.Discard)
	if err != nil {
		return nil, errors.Wrap(err, "parsing versions")
	}

	return newConfig, nil
}
*/