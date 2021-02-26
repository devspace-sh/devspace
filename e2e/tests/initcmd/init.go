package initcmd

import (
	"github.com/loft-sh/devspace/cmd"
	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/spf13/cobra"
)

var _ = ginkgo.Describe("init", func() {
	var (
		f       *customFactory
		testDir string
		tmpDir  string
	)

	ginkgo.BeforeAll(func() {
		var err error
		testDir = "tests/initcmd/testdata"
		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")
		f = &customFactory{
			utils.DefaultFactory,
		}
	})

	ginkgo.AfterAll(func() {
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
	})

	ginkgo.It("Create dockerfile", func() {
		runTest(f, initTestCase{
			dir:     tmpDir + "/data1",
			answers: []string{cmd.CreateDockerfileOption, "go", "Use hub.docker.com => you are logged in as user", "user/data1", "build", "8080", cmd.ComponentChartOption},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Images: map[string]*latest.ImageConfig{
					"app": &latest.ImageConfig{
						Image:                        "user/data1",
						PreferSyncOverRebuild:        true,
						InjectRestartHelper:          true,
						AppendDockerfileInstructions: []string{"USER root"},
						Build: &latest.BuildConfig{
							Docker: &latest.DockerConfig{
								Options: &latest.BuildOptions{
									Target: "build",
								},
							},
						},
					},
				},
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "data1",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "user/data1",
									},
								},
								"service": map[interface{}]interface{}{
									"ports": []interface{}{
										map[interface{}]interface{}{
											"port": 8080,
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
							ImageName: "app",
							PortMappings: []*latest.PortMapping{
								&latest.PortMapping{
									LocalPort: ptr.Int(8080),
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
							ImageName:          "app",
							ExcludePaths:       []string{".git/"},
							UploadExcludePaths: []string{"Dockerfile", "devspace.yaml"},
							OnUpload: &latest.SyncOnUpload{
								RestartContainer: true,
							},
						},
					},
				},
			},
		})
	})

	ginkgo.It("Everything already created", func() {
		runTest(f, initTestCase{
			dir: tmpDir + "/everythingIsThere",
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Images: map[string]*latest.ImageConfig{
					"app": &latest.ImageConfig{
						Image:                        "user/everythingIsThere",
						PreferSyncOverRebuild:        true,
						InjectRestartHelper:          true,
						AppendDockerfileInstructions: []string{"USER root"},
					},
				},
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "everythingIsThere",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "user/everythingIsThere",
									},
								},
								"service": map[interface{}]interface{}{
									"ports": []interface{}{
										map[interface{}]interface{}{
											"port": 8081,
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
							ImageName: "app",
							PortMappings: []*latest.PortMapping{
								&latest.PortMapping{
									LocalPort: ptr.Int(8081),
								},
							},
						},
					},
					Open: []*latest.OpenConfig{
						&latest.OpenConfig{
							URL: "http://localhost:8081",
						},
					},
					Sync: []*latest.SyncConfig{
						&latest.SyncConfig{
							ImageName:          "app",
							ExcludePaths:       []string{".git/"},
							UploadExcludePaths: []string{"Dockerfile", "devspace.yaml"},
							OnUpload: &latest.SyncOnUpload{
								RestartContainer: true,
							},
						},
					},
				},
			},
		})
	})
})

type initTestCase struct {
	dir     string
	answers []string

	expectedConfig *latest.Config
}

func runTest(f *customFactory, testCase initTestCase) {
	err := utils.ChangeWorkingDir(testCase.dir, f.GetLog())
	utils.ExpectNoError(err, "error changing directory")

	initCmd := cmd.InitCmd{
		Dockerfile:  helper.DefaultDockerfilePath,
		Reconfigure: false,
		Context:     "",
		Provider:    "",
	}

	survey, err := f.GetSurvey()
	utils.ExpectNoError(err, "get survey")
	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	err = initCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
	utils.ExpectNoError(err, "executing command")

	if testCase.expectedConfig != nil {
		config, err := f.NewConfigLoader(nil, f.GetLog()).Load()
		utils.ExpectNoError(err, "new config loader")

		utils.ExpectEqual(testCase.expectedConfig, config)
	}
}
