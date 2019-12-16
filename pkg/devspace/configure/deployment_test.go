package configure

/*
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/loader"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"
	"gotest.tools/assert"
)

type GetDockerfileComponentDeploymentTestCase struct {
	name string

	answers           []string
	nameParam         string
	imageName         string
	dockerfile        string
	dockerfileContent string
	context           string

	expectedErr            string
	expectedImage          string
	expectedTag            string
	expectedDockerfile     string
	expectedContext        string
	expectedDeploymentName string
	expectedPort           int
}

func TestGetDockerfileComponentDeployment(t *testing.T) {
	testCases := []GetDockerfileComponentDeploymentTestCase{
		GetDockerfileComponentDeploymentTestCase{
			name:          "Empty params, only answers",
			answers:       []string{"someRegistry.com", "someRegistry.com/user/imagename"},
			expectedImage: "someRegistry.com/user/imagename",
			expectedPort:  1234,
			expectedErr:   "get image config: Registry authentication failed for someRegistry.com/user/imagename.\n         Please login via `docker login someRegistry.com/user/imagename` and try again.",
		},
		GetDockerfileComponentDeploymentTestCase{
			name:               "No answers, only 1 port in dockerfile",
			answers:            []string{},
			imageName:          "someImage",
			dockerfile:         "customDockerFile",
			dockerfileContent:  `EXPOSE 1010`,
			expectedImage:      "someImage",
			expectedDockerfile: "customDockerFile",
			expectedPort:       1010,
		},
		GetDockerfileComponentDeploymentTestCase{
			name:       "2 ports in dockerfile",
			answers:    []string{""},
			imageName:  "someImage",
			dockerfile: "customDockerFile",
			dockerfileContent: `EXPOSE 1011
EXPOSE 1012`,
			expectedImage:      "someImage",
			expectedDockerfile: "customDockerFile",
			expectedPort:       1011,
		},
	}

	//Create tempDir and go into it
	dir, err := ioutil.TempDir("", "testDir")
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

	// Delete temp folder after test
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	for _, testCase := range testCases {
		testConfig := &latest.Config{}
		generated := &generated.Config{}

		if testCase.dockerfile != "" {
			err = fsutil.WriteToFile([]byte(testCase.dockerfileContent), testCase.dockerfile)
			assert.NilError(t, err, "Error overwriting dockerfile in testCase %s", testCase.name)
		}

		for _, answer := range testCase.answers {
			survey.SetNextAnswer(answer)
		}

		imageConfig, deploymentConfig, err := GetDockerfileComponentDeployment(testConfig, generated, testCase.nameParam, testCase.imageName, testCase.dockerfile, testCase.context, log.GetInstance())

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Image, testCase.expectedImage, "Returned image is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Tag, testCase.expectedTag, "Returned tag is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Dockerfile, testCase.expectedDockerfile, "Returned dockerfile is unexpected in testCase %s", testCase.name)
			assert.Equal(t, imageConfig.Context, testCase.expectedContext, "Returned context is unexpected in testCase %s", testCase.name)
			assert.Equal(t, deploymentConfig.Name, testCase.expectedDeploymentName, "Returned deployment name is unexpected in testCase %s", testCase.name)
			// assert.Equal(t, *deploymentConfig.Component.Service.Ports[0].Port, testCase.expectedPort, "Returned port in deployment is unexpected in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type getImageComponentDeploymentTestCase struct {
	name string

	answers   []string
	icdName   string
	imageName string

	expectedNilImage       bool
	expectedErr            string
	expectedImage          string
	expectedTag            string
	expectedDockerfile     string
	expectedContext        string
	expectedDeploymentName string
	expectedPort           int
}

func TestGetImageComponentDeployment(t *testing.T) {
	testCases := []getImageComponentDeploymentTestCase{
		getImageComponentDeploymentTestCase{
			name:      "valid with port",
			answers:   []string{"12345"},
			icdName:   "someDeployment",
			imageName: "someImage:someTag",

			expectedImage:          "someImage",
			expectedTag:            "someTag",
			expectedDeploymentName: "someDeployment",
			expectedPort:           12345,
		},
	}

	for _, testCase := range testCases {
		for _, answer := range testCase.answers {
			survey.SetNextAnswer(answer)
		}

		imageConfig, deploymentConfig, err := GetImageComponentDeployment(testCase.icdName, testCase.imageName, log.GetInstance())

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			if !testCase.expectedNilImage {
				if imageConfig == nil {
					t.Fatalf("Nil image returned in testCase %s", testCase.name)
				}
				assert.Equal(t, imageConfig.Image, testCase.expectedImage, "Returned image is unexpected in testCase %s", testCase.name)
				assert.Equal(t, imageConfig.Tag, testCase.expectedTag, "Returned tag is unexpected in testCase %s", testCase.name)
				assert.Equal(t, imageConfig.Dockerfile, testCase.expectedDockerfile, "Returned dockerfile is unexpected in testCase %s", testCase.name)
				assert.Equal(t, imageConfig.Context, testCase.expectedContext, "Returned context is unexpected in testCase %s", testCase.name)
			} else {
				if imageConfig != nil {
					t.Fatalf("Not nil returned in testCase %s", testCase.name)
				}
			}
			assert.Equal(t, deploymentConfig.Name, testCase.expectedDeploymentName, "Returned deployment name is unexpected in testCase %s", testCase.name)
			//assert.Equal(t, *deploymentConfig.Component.Service.Ports[0].Port, testCase.expectedPort, "Returned port in deployment is unexpected in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type getPredefinedComponentDeploymentTestCase struct {
	name string

	answers        []string
	deploymentName string
	componentName  string

	expectedErr            string
	expectedDeploymentName string
	expectedPort           int
}

func TestGetPredefinedComponentDeployment(t *testing.T) {
	testCases := []getPredefinedComponentDeploymentTestCase{}

	for _, testCase := range testCases {
		for _, answer := range testCase.answers {
			survey.SetNextAnswer(answer)
		}

		deploymentConfig, err := GetPredefinedComponentDeployment(testCase.deploymentName, testCase.componentName, log.GetInstance())

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			assert.Equal(t, deploymentConfig.Name, testCase.expectedDeploymentName, "Returned deployment name is unexpected in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type getKubectlDeploymentTestCase struct {
	name string

	deploymentName string
	manifests      string

	expectedErr             string
	expectedDeploymentName  string
	expectedSplittedPointer []string
}

func TestGetKubectlDeployment(t *testing.T) {
	testCases := []getKubectlDeploymentTestCase{
		getKubectlDeploymentTestCase{
			name:                    "Valid and only testCase",
			deploymentName:          "someDeployment",
			manifests:               "these, are , some   , mani fests ",
			expectedDeploymentName:  "someDeployment",
			expectedSplittedPointer: []string{"these", "are", "some", "mani fests"},
		},
	}

	for _, testCase := range testCases {
		deploymentConfig, err := GetKubectlDeployment(testCase.deploymentName, testCase.manifests)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			assert.Equal(t, deploymentConfig.Name, testCase.expectedDeploymentName, "Returned deployment name is unexpected in testCase %s", testCase.name)
			assert.Equal(t, len(deploymentConfig.Kubectl.Manifests), len(testCase.expectedSplittedPointer), "Returned manifest has unexpected length in testCase %s", testCase.name)
			for index, expected := range testCase.expectedSplittedPointer {
				assert.Equal(t, deploymentConfig.Kubectl.Manifests[index], expected, "Returned manifest in index %d is unexpected in testCase %s", index, testCase.name)
			}
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type getHelmDeploymentTestCase struct {
	name string

	deploymentName string
	chartName      string
	chartRepo      string
	chartVersion   string

	expectedErr              string
	expectedDeploymentName   string
	expectedHelmChartName    string
	expectedHelmChartRepo    string
	expectedHelmChartVersion string
}

func TestGetHelmDeployment(t *testing.T) {
	testCases := []getHelmDeploymentTestCase{
		getHelmDeploymentTestCase{
			name:                     "Valid and only testCase",
			deploymentName:           "someDeployment",
			chartName:                "someChart",
			chartRepo:                "someChartRepo",
			chartVersion:             "someChartVersion",
			expectedDeploymentName:   "someDeployment",
			expectedHelmChartName:    "someChart",
			expectedHelmChartRepo:    "someChartRepo",
			expectedHelmChartVersion: "someChartVersion",
		},
	}

	for _, testCase := range testCases {
		deploymentConfig, err := GetHelmDeployment(testCase.deploymentName, testCase.chartName, testCase.chartRepo, testCase.chartVersion)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
			assert.Equal(t, deploymentConfig.Name, testCase.expectedDeploymentName, "Returned deployment name is unexpected in testCase %s", testCase.name)
			assert.Equal(t, deploymentConfig.Helm.Chart.Name, testCase.expectedHelmChartName, "Returned chart name is unexpected in testCase %s", testCase.name)
			assert.Equal(t, deploymentConfig.Helm.Chart.RepoURL, testCase.expectedHelmChartRepo, "Returned chart name is unexpected in testCase %s", testCase.name)
			assert.Equal(t, deploymentConfig.Helm.Chart.Version, testCase.expectedHelmChartVersion, "Returned chart name is unexpected in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}

type removeDeploymentTestCase struct {
	name string

	deploymentName      string
	allFlag             bool
	existingDeployments []*latest.DeploymentConfig

	expectedErr                  string
	expectedFound                bool
	expectedRemainingDeployments []string
}

func TestRemoveDeployment(t *testing.T) {
	testCases := []removeDeploymentTestCase{
		removeDeploymentTestCase{
			name:        "Invalid input",
			expectedErr: "You have to specify either a deployment name or the --all flag",
		},
		removeDeploymentTestCase{
			name:    "Remove all 2 deployments",
			allFlag: true,
			existingDeployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: "someDeploy",
					Helm: &latest.HelmConfig{},
				},
				&latest.DeploymentConfig{
					Name: "otherDeploy",
					Helm: &latest.HelmConfig{},
				},
			},
			expectedFound: true,
		},
		removeDeploymentTestCase{
			name:                "Remove all 0 deployments",
			allFlag:             true,
			existingDeployments: []*latest.DeploymentConfig{},
			expectedFound:       false,
		},
		removeDeploymentTestCase{
			name:           "Remove 1 of 2 deployments",
			deploymentName: "someDeploy",
			existingDeployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: "someDeploy",
					Helm: &latest.HelmConfig{},
				},
				&latest.DeploymentConfig{
					Name: "otherDeploy",
					Helm: &latest.HelmConfig{},
				},
			},
			expectedFound:                true,
			expectedRemainingDeployments: []string{"otherDeploy"},
		},
		removeDeploymentTestCase{
			name:           "Remove 1 deployment that does not exist",
			deploymentName: "notExistent",
			existingDeployments: []*latest.DeploymentConfig{
				&latest.DeploymentConfig{
					Name: "someDeploy",
					Helm: &latest.HelmConfig{},
				},
				&latest.DeploymentConfig{
					Name: "otherDeploy",
					Helm: &latest.HelmConfig{},
				},
			},
			expectedFound:                false,
			expectedRemainingDeployments: []string{"someDeploy", "otherDeploy"},
		},
	}

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

	//Delete temp folder
	defer func() {
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	for _, testCase := range testCases {
		fakeConfig := &latest.Config{}
		if testCase.existingDeployments != nil {
			fakeConfig.Deployments = testCase.existingDeployments
		}
		loader.SetFakeConfig(fakeConfig)

		config, err := loader.GetBaseConfig(nil)
		if err != nil {
			log.Fatal(err)
		}

		found, err := RemoveDeployment(config, testCase.allFlag, testCase.deploymentName)

		assert.Equal(t, found, testCase.expectedFound, "Returned found-boolean unexpected in testCase %s", testCase.name)
		assert.Equal(t, len(fakeConfig.Deployments), len(testCase.expectedRemainingDeployments), "Unexpected amount of remaining deployments in testCase %s", testCase.name)
		for index, expectedName := range testCase.expectedRemainingDeployments {
			assert.Equal(t, fakeConfig.Deployments[index].Name, expectedName, "Unexpected remaining deployment in testCase %s", testCase.name)
		}

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}
	}
}*/
