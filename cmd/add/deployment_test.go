package add

/* @Florian adjust to new behaviour
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/configutil"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/constants"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type addDeploymentTestCase struct {
	name string

	fakeConfig    *latest.Config
	fakeGenerated *generated.Config

	args            []string
	answers         []string
	cmdManifests    string
	cmdChart        string
	cmdChartRepo    string
	cmdChartVersion string
	cmdDockerfile   string
	cmdImage        string
	cmdContext      string
	cmdNamespace    string
	componentFlag   string

	expectedOutput              string
	expectedPanic               string
	expectConfigFile            bool
	expectedDeploymentName      string
	expectedDeploymentNamespace string
	expectedDeploymentsNumber   int

	expectedDeploymentKubectlManifests []string

	expectedHelmChartName    string
	expectedHelmChartRepoURL string
	expectedHelmChartVersion string

	expectedImagesNumber           int
	expectedImageName              string
	expectedImageTag               string
	expectedImageDockerfile        string
	expectedImageContext           string
	expectedImageCreatePullSecrets bool
}

func TestRunAddDeployment(t *testing.T) {
	testCases := []addDeploymentTestCase{
		addDeploymentTestCase{
			name:          "No devspace config",
			args:          []string{""},
			expectedPanic: "Couldn't find a DevSpace configuration. Please run `devspace init`",
		},
		addDeploymentTestCase{
			name:          "No params",
			args:          []string{""},
			fakeConfig:    &latest.Config{},
			expectedPanic: "Please specifiy one of these parameters:\n--image: A docker image to deploy (e.g. dscr.io/myuser/myrepo or dockeruser/repo:0.1 or mysql:latest)\n--manifests: The kubernetes manifests to deploy (glob pattern are allowed, comma separated, e.g. manifests/** or kube/pod.yaml)\n--chart: A helm chart to deploy (e.g. ./chart or stable/mysql)\n--component: A predefined component to use (run `devspace list available-components` to see all available components)",
		},
		addDeploymentTestCase{
			name: "Add already existing deployment",
			args: []string{"exists"},
			fakeConfig: &latest.Config{
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "exists",
					},
				},
			},
			expectedPanic: "Deployment exists already exists",
		},
		addDeploymentTestCase{
			name: "Valid kubectl deployment",
			args: []string{"newKubectlDeployment"},
			fakeConfig: &latest.Config{
				Version: "v1beta3",
			},

			cmdManifests: "these, are, manifests",
			cmdNamespace: "kubectlNamespace",

			expectedOutput:                     "\nDone Successfully added newKubectlDeployment as new deployment",
			expectConfigFile:                   true,
			expectedDeploymentsNumber:          1,
			expectedDeploymentName:             "newKubectlDeployment",
			expectedDeploymentNamespace:        "kubectlNamespace",
			expectedDeploymentKubectlManifests: []string{"these", "are", "manifests"},
		},
		addDeploymentTestCase{
			name: "Valid helm deployment",
			args: []string{"newHelmDeployment"},
			fakeConfig: &latest.Config{
				Version: "v1beta3",
			},

			cmdChart:        "myChart",
			cmdChartRepo:    "myChartRepo",
			cmdChartVersion: "myChartVersion",

			expectedOutput:            "\nDone Successfully added newHelmDeployment as new deployment",
			expectConfigFile:          true,
			expectedDeploymentsNumber: 1,
			expectedDeploymentName:    "newHelmDeployment",
			expectedHelmChartName:     "myChart",
			expectedHelmChartRepoURL:  "myChartRepo",
			expectedHelmChartVersion:  "myChartVersion",
		},
		addDeploymentTestCase{
			name:    "Valid dockerfile deployment",
			args:    []string{"newDockerfileDeployment"},
			answers: []string{"1234"},
			fakeConfig: &latest.Config{
				Version: "v1beta3",
			},

			cmdDockerfile: "myDockerfile",
			cmdImage:      "myImage",
			cmdContext:    "myContext",

			expectedOutput:            "\nDone Successfully added newDockerfileDeployment as new deployment",
			expectConfigFile:          true,
			expectedDeploymentsNumber: 1,
			expectedDeploymentName:    "newDockerfileDeployment",

			expectedImagesNumber:           1,
			expectedImageName:              "myImage",
			expectedImageDockerfile:        "myDockerfile",
			expectedImageContext:           "myContext",
			expectedImageCreatePullSecrets: true,
		},
		addDeploymentTestCase{
			name:    "Valid image deployment",
			args:    []string{"newImageDeployment"},
			answers: []string{"1234"},
			fakeConfig: &latest.Config{
				Version: "v1beta3",
				Images: map[string]*latest.ImageConfig{
					"someImage": &latest.ImageConfig{
						Image: "someImage",
					},
				},
			},

			cmdImage:   "myImage",
			cmdContext: "myContext",

			expectedOutput:            "\nDone Successfully added newImageDeployment as new deployment",
			expectConfigFile:          true,
			expectedDeploymentsNumber: 1,
			expectedDeploymentName:    "newImageDeployment",

			expectedImagesNumber:           2,
			expectedImageName:              "myImage",
			expectedImageTag:               "latest",
			expectedImageCreatePullSecrets: true,
		},
	}

	log.SetInstance(&testLogger{
		log.DiscardLogger{PanicOnExit: true},
	})

	for _, testCase := range testCases {
		testRunAddDeployment(t, testCase)
	}
}

func testRunAddDeployment(t *testing.T, testCase addDeploymentTestCase) {
	logOutput = ""

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

	isDeploymentsNil := testCase.fakeConfig == nil || testCase.fakeConfig.Deployments == nil
	configutil.SetFakeConfig(testCase.fakeConfig)
	if isDeploymentsNil && testCase.fakeConfig != nil {
		testCase.fakeConfig.Deployments = nil
	}

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

		rec := recover()
		if testCase.expectedPanic == "" {
			if rec != nil {
				t.Fatalf("Unexpected panic in testCase %s. Message: %s", testCase.name, rec)
			}
		} else {
			if rec == nil {
				t.Fatalf("Unexpected no panic in testCase %s", testCase.name)
			} else {
				assert.Equal(t, rec, testCase.expectedPanic, "Wrong panic message in testCase %s", testCase.name)
			}
		}
		assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)
	}()

	(&deploymentCmd{
		Manifests:    testCase.cmdManifests,
		Chart:        testCase.cmdChart,
		ChartRepo:    testCase.cmdChartRepo,
		ChartVersion: testCase.cmdChartVersion,
		Dockerfile:   testCase.cmdDockerfile,
		Image:        testCase.cmdImage,
		Context:      testCase.cmdContext,
		Namespace:    testCase.cmdNamespace,
		Component:    testCase.componentFlag,
	}).RunAddDeployment(nil, testCase.args)

	assert.Equal(t, logOutput, testCase.expectedOutput, "Unexpected output in testCase %s", testCase.name)

	if testCase.expectConfigFile {
		config, err := loadConfigFromPath()
		assert.NilError(t, err, "Error loading config after adding deployment in testCase %s. Maybe it was unexpectedly not saved in %s", testCase.name, constants.DefaultConfigPath)

		assert.Equal(t, len(config.Deployments), testCase.expectedDeploymentsNumber, "Unexpected number of deployments in testCase %s", testCase.name)
		if testCase.expectedDeploymentsNumber != 0 {
			assert.Equal(t, config.Deployments[0].Name, testCase.expectedDeploymentName, "Unexpected name of new deployment in testCase %s", testCase.name)
			assert.Equal(t, config.Deployments[0].Namespace, testCase.expectedDeploymentNamespace, "Unexpected name of new deployment in testCase %s", testCase.name)

			if len(testCase.expectedDeploymentKubectlManifests) != 0 {
				assert.Equal(t, config.Deployments[0].Kubectl == nil || config.Deployments[0].Kubectl.Manifests == nil, false, "Kubectl manifests are unexpectedly nil in testCase %s", testCase.name)
				assert.Equal(t, len(config.Deployments[0].Kubectl.Manifests), len(testCase.expectedDeploymentKubectlManifests), "Returned manifest has unexpected length in testCase %s", testCase.name)
				for index, expected := range testCase.expectedDeploymentKubectlManifests {
					assert.Equal(t, config.Deployments[0].Kubectl.Manifests[index], expected, "Returned manifest in index %d is unexpected in testCase %s", index, testCase.name)
				}
			} else if testCase.expectedHelmChartName != "" {
				assert.Equal(t, config.Deployments[0].Helm == nil || config.Deployments[0].Helm.Chart == nil, false, "Helm field is unexpectedly nil in testCase %s", testCase.name)
				assert.Equal(t, config.Deployments[0].Helm.Chart.Name, testCase.expectedHelmChartName, "Helm chart of new deployment has wrong name in testCase %s", testCase.name)
				assert.Equal(t, config.Deployments[0].Helm.Chart.RepoURL, testCase.expectedHelmChartRepoURL, "Helm chart of new deployment has wrong RepoURL in testCase %s", testCase.name)
				assert.Equal(t, config.Deployments[0].Helm.Chart.Version, testCase.expectedHelmChartVersion, "Helm chart of new deployment has wrong version in testCase %s", testCase.name)
			}
		}

		assert.Equal(t, len(config.Images), testCase.expectedImagesNumber, "Unexpected number of images in testCase %s", testCase.name)
		if testCase.expectedImagesNumber != 0 {
			assert.Equal(t, config.Images[testCase.expectedDeploymentName] == nil, false, "No image with expected name in testCase %s", testCase.name)
			assert.Equal(t, config.Images[testCase.expectedDeploymentName].Image, testCase.expectedImageName, "Image has unexpected name in testCase %s", testCase.name)
			assert.Equal(t, config.Images[testCase.expectedDeploymentName].Tag, testCase.expectedImageTag, "Image has unexpected tag in testCase %s", testCase.name)
			assert.Equal(t, config.Images[testCase.expectedDeploymentName].Dockerfile, testCase.expectedImageDockerfile, "Image has unexpected dockerfile in testCase %s", testCase.name)
			assert.Equal(t, config.Images[testCase.expectedDeploymentName].Context, testCase.expectedImageContext, "Image has unexpected context in testCase %s", testCase.name)
			assert.Equal(t, ptr.ReverseBool(config.Images[testCase.expectedDeploymentName].CreatePullSecret), testCase.expectedImageCreatePullSecrets, "Image has unexpected pull secrets settings name in testCase %s", testCase.name)
		}
	}

	err = os.Remove(constants.DefaultConfigPath)
	assert.Equal(t, !os.IsNotExist(err), testCase.expectConfigFile, "Unexpectedly saved or not saved in testCase %s", testCase.name)

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

	newConfig, err := versions.Parse(oldConfig)
	if err != nil {
		return nil, err
	}

	return newConfig, nil
}
*/
