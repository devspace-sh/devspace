package loader

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	config2 "github.com/loft-sh/devspace/pkg/devspace/config"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/remotecache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	fakekubectl "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/log"
	yaml "gopkg.in/yaml.v3"
	"gotest.tools/assert"
)

type existsTestCase struct {
	name string

	files      map[string]interface{}
	configPath string

	expectedanswer bool
}

func TestExists(t *testing.T) {
	testCases := []existsTestCase{
		{
			name:       "Only custom file name exists",
			configPath: "mypath.yaml",
			files: map[string]interface{}{
				"mypath.yaml": "",
			},
			expectedanswer: true,
		},
		{
			name: "Default file name does not exist",
			files: map[string]interface{}{
				"mypath.yaml": "",
			},
			expectedanswer: false,
		},
	}

	dir := t.TempDir()

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
	}()

	for _, testCase := range testCases {
		testExists(testCase, t)
	}
}

func testExists(testCase existsTestCase, t *testing.T) {
	defer func() {
		for _, path := range []string{".devspace/generated.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}

	loader := &configLoader{
		absConfigPath: testCase.configPath,
	}

	exists := loader.Exists()
	assert.Equal(t, exists, testCase.expectedanswer, "Unexpected answer in testCase %s", testCase.name)
}

type cloneTestCase struct {
	name string

	cloner ConfigOptions

	expectedClone *ConfigOptions
	expectedErr   string
}

func TestClone(t *testing.T) {
	testCases := []cloneTestCase{
		{
			name: "Clone ConfigOptions",
			cloner: ConfigOptions{
				Profiles: []string{"clonerProf"},
			},
			expectedClone: &ConfigOptions{
				Profiles: []string{"clonerProf"},
			},
		},
	}

	for _, testCase := range testCases {
		clone, err := (&testCase.cloner).Clone()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		configAsYaml, err := yaml.Marshal(clone)
		assert.NilError(t, err, "Error parsing clone in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedClone)
		assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected clone in testCase %s", testCase.name)
	}
}

type loadTestCase struct {
	name string

	configPath        string
	options           ConfigOptions
	returnedGenerated localcache.LocalCache
	files             map[string]interface{}
	withProfile       bool

	expectedConfig *latest.Config
	expectedErr    string
}

func TestLoad(t *testing.T) {
	testCases := []*loadTestCase{
		{
			name:       "Get from custom config file with profile",
			configPath: "custom.yaml",
			options:    ConfigOptions{},
			files: map[string]interface{}{
				"custom.yaml": latest.Config{
					Version: latest.Version,
					Profiles: []*latest.ProfileConfig{
						{
							Name: "active",
						},
					},
				},
			},
			returnedGenerated: localcache.LocalCache{},
			withProfile:       true,
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Name:    "devspace",
				Dev:     latest.NewRaw().Dev,
			},
		},
		{
			name:    "Get from default file without profile",
			options: ConfigOptions{},
			files: map[string]interface{}{
				"devspace.yaml": latest.Config{
					Version: latest.Version,
					Name:    "devspace",
				},
			},
			expectedConfig: &latest.Config{
				Version: latest.Version,
				Name:    "devspace",
				Dev:     latest.NewRaw().Dev,
			},
		},
	}

	dir := t.TempDir()

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
	}()

	for _, testCase := range testCases {
		testLoad(testCase, t)
	}
}

func testLoad(testCase *loadTestCase, t *testing.T) {
	defer func() {
		for _, path := range []string{".devspace/generated.yaml", "devspace.yaml", "custom.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}

	loader := &configLoader{
		absConfigPath: testCase.configPath,
	}

	var config config2.Config
	var err error
	config, err = loader.Load(context.TODO(), nil, &testCase.options, log.Discard)
	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	configAsYaml, err := yaml.Marshal(config.Config())
	assert.NilError(t, err, "Error parsing config in testCase %s", testCase.name)
	expectedAsYaml, err := yaml.Marshal(testCase.expectedConfig)
	assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
	assert.Equal(t, string(configAsYaml), string(expectedAsYaml), "Unexpected config in testCase %s", testCase.name)
}

type setDevSpaceRootTestCase struct {
	name string

	configPath string
	startDir   string
	files      map[string]interface{}

	expectedExists     bool
	expectedWorkDir    string
	expectedConfigPath string
	expectedErr        string
}

func TestSetDevSpaceRoot(t *testing.T) {
	dir := t.TempDir()

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
	}()

	testCases := []setDevSpaceRootTestCase{
		{
			name:       "No custom.yaml",
			configPath: "custom.yaml",
			files: map[string]interface{}{
				"devspace.yaml": "",
			},
			expectedExists:     false,
			expectedWorkDir:    dir,
			expectedConfigPath: "custom.yaml",
		},
		{
			name:            "No devspace.yaml",
			expectedExists:  false,
			expectedWorkDir: dir,
		},
		{
			name: "Config exists",
			files: map[string]interface{}{
				"devspace.yaml": "",
			},
			startDir:        "subDir",
			expectedExists:  true,
			expectedWorkDir: dir,
		},
		{
			name:       "Custom config in subdir exists",
			configPath: "subdir/custom.yaml",
			files: map[string]interface{}{
				"subdir/custom.yaml": "",
			},
			expectedExists: true,
			expectedWorkDir: func() string {
				if runtime.GOOS == "darwin" {
					return filepath.Join(dir, "subDir")
				}
				return filepath.Join(dir, "subdir")
			}(),
			expectedConfigPath: "subdir/custom.yaml",
		},
	}

	for _, testCase := range testCases {
		testSetDevSpaceRoot(testCase, t)
	}
}

func testSetDevSpaceRoot(testCase setDevSpaceRootTestCase, t *testing.T) {
	wdBackup, err := os.Getwd()
	assert.NilError(t, err, "Error getting current working directory")
	defer func() {
		_ = os.Chdir(wdBackup)
		for _, path := range []string{"devspace.yaml", "custom.yaml"} {
			_ = os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}
	if testCase.startDir != "" {
		_ = os.Mkdir(testCase.startDir, os.ModePerm)
		_ = os.Chdir(testCase.startDir)
	}

	loader := &configLoader{
		absConfigPath: testCase.configPath,
	}

	exists, err := loader.SetDevSpaceRoot(log.Discard)

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}
	assert.Equal(t, exists, testCase.expectedExists, "Unexpected existence answer in testCase %s", testCase.name)

	wd, err := os.Getwd()
	if runtime.GOOS == "darwin" {
		wd = strings.ReplaceAll(wd, "/private", "")
	}
	assert.NilError(t, err, "Error getting wd in testCase %s", testCase.name)
	assert.Equal(t, wd, testCase.expectedWorkDir, "Unexpected work dir in testCase %s", testCase.name)

	assert.Equal(t, loader.absConfigPath, testCase.expectedConfigPath, "Unexpected configPath in testCase %s", testCase.name)
}

type getProfilesTestCase struct {
	name string

	files      map[string]interface{}
	configPath string

	expectedProfiles []string
	expectedErr      string
}

func TestGetProfiles(t *testing.T) {
	testCases := []getProfilesTestCase{
		{
			name: "Empty file",
			files: map[string]interface{}{
				"devspace.yaml": map[string]interface{}{
					"version": "v1beta9",
				},
			},
		},
		{
			name:       "Parse several profiles",
			configPath: "custom.yaml",
			files: map[string]interface{}{
				"custom.yaml": map[string]interface{}{
					"version": "v1beta9",
					"profiles": []interface{}{
						map[string]interface{}{
							"name": "myprofile",
						},
					},
				},
			},
			expectedProfiles: []string{"myprofile"},
		},
	}

	dir := t.TempDir()

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	defer func() {
		//Delete temp folder
		err = os.Chdir(wdBackup)
		if err != nil {
			t.Fatalf("Error changing dir back: %v", err)
		}
	}()

	for _, testCase := range testCases {
		testGetProfiles(testCase, t)
	}
}

func testGetProfiles(testCase getProfilesTestCase, t *testing.T) {
	defer func() {
		for _, path := range []string{".devspace/generated.yaml", "devspace.yaml", "custom.yaml"} {
			os.Remove(path)
		}
	}()
	for path, data := range testCase.files {
		dataAsYaml, err := yaml.Marshal(data)
		assert.NilError(t, err, "Error parsing data of file %s in testCase %s", path, testCase.name)
		err = fsutil.WriteToFile([]byte(dataAsYaml), path)
		assert.NilError(t, err, "Error writing file %s in testCase %s", path, testCase.name)
	}

	loader := &configLoader{
		absConfigPath: testCase.configPath,
	}
	c, err := loader.LoadWithParser(context.Background(),
		localcache.New(constants.DefaultCacheFolder),
		&fakekubectl.Client{Client: fake.NewSimpleClientset()},
		NewProfilesParser(),
		&ConfigOptions{},
		log.Discard)
	assert.NilError(t, err, "Error loading config in testCase %s", testCase.name)
	profileObjects := c.Config().Profiles
	profiles := []string{}
	for _, p := range profileObjects {
		profiles = append(profiles, p.Name)
	}

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Error in testCase %s", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
	}

	assert.Equal(t, strings.Join(profiles, ","), strings.Join(testCase.expectedProfiles, ","), "Unexpected profiles in testCase %s", testCase.name)
}

type parseCommandsTestCase struct {
	name string

	data map[string]interface{}

	expectedCommands map[string]*latest.CommandConfig
	expectedErr      string
}

// TODO: Finish this test!
func TestParseCommands(t *testing.T) {
	testCases := []parseCommandsTestCase{
		{
			data: map[string]interface{}{
				"version": latest.Version,
			},
		},
	}

	for idx, testCase := range testCases {
		t.Run("Test "+strconv.Itoa(idx), func(t *testing.T) {
			f, err := os.CreateTemp("", "")
			if err != nil {
				t.Fatal(err)
			}

			defer os.Remove(f.Name())

			out, err := yaml.Marshal(testCase.data)
			if err != nil {
				t.Fatal(err)
			}

			_, err = f.Write(out)
			if err != nil {
				t.Fatal(err)
			}

			// Close before reading
			f.Close()
			loader := &configLoader{
				absConfigPath: f.Name(),
			}

			commandsInterface, err := loader.LoadWithParser(context.Background(),
				localcache.New(constants.DefaultConfigPath),
				&fakekubectl.Client{Client: fake.NewSimpleClientset()},
				NewCommandsParser(),
				nil, log.Discard)
			if testCase.expectedErr == "" {
				assert.NilError(t, err, "Error in testCase %s", testCase.name)
			} else {
				assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
			}
			commands := commandsInterface.Config().Commands

			commandsAsYaml, err := yaml.Marshal(commands)
			assert.NilError(t, err, "Error parsing commands in testCase %s", testCase.name)
			expectedAsYaml, err := yaml.Marshal(testCase.expectedCommands)
			assert.NilError(t, err, "Error parsing exception to yaml in testCase %s", testCase.name)
			assert.Equal(t, string(commandsAsYaml), string(expectedAsYaml), "Unexpected commands in testCase %s", testCase.name)
		})
	}
}

type parseTestCase struct {
	name        string
	in          *parseTestCaseInput
	expected    *latest.Config
	expectedErr string
}

// TODO: only for lint purpose, remove once the below test is fixed
var _ = parseTestCase{
	in:          &parseTestCaseInput{},
	expected:    &latest.Config{},
	expectedErr: "",
}

type parseTestCaseInput struct {
	config          string
	options         *ConfigOptions
	generatedConfig *localcache.LocalCache
}

// TODO: only for lint purpose, remove once the below test is fixed
var _ = parseTestCaseInput{
	config:          "",
	options:         &ConfigOptions{},
	generatedConfig: &localcache.LocalCache{},
}

var convertedVars = map[string]*latest.Variable{
	"DEVSPACE_ENV_FILE": &latest.Variable{
		Value:         ".env",
		AlwaysResolve: &[]bool{false}[0],
	},
}

var testPipeline = map[string]*latest.Pipeline{
	"build": &latest.Pipeline{},
	"deploy": &latest.Pipeline{
		Run: "create_deployments test --sequential",
	},
	"dev": &latest.Pipeline{
		Run: `create_deployments test --sequential

start_dev --all`,
	},
	"purge": &latest.Pipeline{
		Run: `stop_dev --all
purge_deployments test --sequential`,
	},
}

func TestParseConfig(t *testing.T) {
	testCases := []*parseTestCase{
		{
			name: "Simple",
			in: &parseTestCaseInput{
				config: `
version: v1beta11`,
				options:         &ConfigOptions{},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Pipelines: map[string]*latest.Pipeline{
					"build":  &latest.Pipeline{},
					"deploy": &latest.Pipeline{},
					"dev": &latest.Pipeline{
						Run: "start_dev --all",
					},
					"purge": &latest.Pipeline{
						Run: "stop_dev --all",
					},
				},
				Vars: convertedVars,
			},
		},
		{
			name: "Simple with deployments",
			in: &parseTestCaseInput{
				config: `
version: v1beta3
deployments:
- name: test
  component:
    containers:
    - image: nginx`,
				options:         &ConfigOptions{},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
				Vars: convertedVars,
			},
		},
		{
			name: "Variables",
			in: &parseTestCaseInput{
				config: `
version: v1beta3
vars:
- name: my_var
deployments:
- name: ${my_var}
  component:
    containers:
    - image: nginx`,
				options: &ConfigOptions{},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{
					"my_var": "test",
				}},
			},
			expected: &latest.Config{
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
				Vars: convertedVars,
			},
		},
		{
			name: "Profile replace with variable",
			in: &parseTestCaseInput{
				config: `version: v1beta11
vars:
- name: test_var
deployments:
- name: ${does-not-exist}
  helm:
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  replace:
    deployments:
    - name: ${test_var}
      helm:
        values:
          containers:
          - image: ubuntu
`,
				options: &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{
					"test_var": "test",
				}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "ubuntu",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profiles defined with expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles: |-
  $(echo """
  - name: testprofile
    replace:
      deployments:
      - name: test
        helm:
          componentChart: true
          values:
            containers:
            - image: ubuntu
  """)`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Pipelines: testPipeline,
				Vars:      convertedVars,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "ubuntu",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profile with merge expression",
			in: &parseTestCaseInput{
				config: `
		version: v1beta11
		deployments:
		- name: deployment
		  helm:
		    componentChart: true
		    values:
		      containers:
		      - image: nginx
		profiles:
		- name: testprofile
		  merge: |-
		    $(echo """
		    deployments:
		      - name: test
		        helm:
		          componentChart: true
		          values:
		            containers:
		            - image: ubuntu
		    """)
		`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "ubuntu",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profile with replace expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  replace: |-
    $(echo """
    deployments:
      - name: test
        helm:
          componentChart: true
          values:
            containers:
            - image: ubuntu
    """)
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "ubuntu",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profile with parent expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testparent
- name: testprofile
  parent: $(echo testparent)
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expectedErr: `error validating profiles[1]: parent cannot be an expression`,
		},
		{
			name: "Profile with parent variable",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testparent
- name: testprofile
  parent: ${testparent}
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"testparent": "testparent"}},
			},
			expectedErr: `error validating profiles[1]: parent cannot be a variable`,
		},
		{
			name: "Profile with parents expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testparent
- name: testprofile
  parents: $(echo [testparent])
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expectedErr: `error validating profiles[1]: parents cannot be an expression`,
		},
		{
			name: "Profile with parents variable",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testparent
- name: testprofile
  parents: ${testparents}
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"testparent": "testparent"}},
			},
			expectedErr: `error validating profiles[1]: parents cannot be a variable`,
		},
		{
			name: "Profile with activations expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  activation: $(echo [testparent])
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expectedErr: `error validating profiles[0]: activation cannot be an expression`,
		},
		{
			name: "Profile with activations variable",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  activation: ${testparents}
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"testparent": "testparent"}},
			},
			expectedErr: `error validating profiles[0]: activation cannot be a variable`,
		},
		{
			name: "Profile with patches expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  patches: |-
    $(echo """
    - path: deployments
      op: replace
      value:
      - name: test
        helm:
          componentChart: true
          values:
            containers:
            - image: ubuntu
    """)
`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "ubuntu",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profile with patch op variable",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  patches:
  - path: deployments
    op: ${op}
    value:
      - name: deployment
        helm:
          componentChart: true
          values:
            containers:
            - image: ubuntu
`,
				options: &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{
					"op": "replace",
				}},
			},
			expectedErr: "error validating profiles[0]: patches[0] op cannot be a variable",
		},
		{
			name: "Profile with patch op expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  patches:
  - path: deployments
    op: $(echo replace)
    value:
      - name: deployment
        helm:
          componentChart: true
          values:
            containers:
            - image: ubuntu
`,
				options: &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{
					"path": "deployments",
				}},
			},
			expectedErr: "error validating profiles[0]: patches[0] op cannot be an expression",
		},
		{
			name: "Profile with patch value variable",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
deployments:
- name: test
  helm:
    componentChart: true
    valuesFiles:
      - test.yaml
profiles:
- name: testprofile
  patches:
  - path: deployments..valuesFiles[0]
    op: replace
    value: $(echo ubuntu)
`,
				options: &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{
					"test_var": "test",
				}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							ValuesFiles: []string{"ubuntu"},
						},
					},
				},
			},
		},
		{
			name: "Profile with patch value expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
vars:
- name: IMAGE
  default: ubuntu
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: testprofile
  patches:
  - path: deployments
    op: replace
    value: |-
      $(echo """
      - name: test
        helm:
          componentChart: true
          values:
            containers:
            - image: ${IMAGE}
      """)
`,
				options: &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{
					"IMAGE": "foo",
				}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "foo",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profiles with variables ignored when not activated",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
vars:
- name: IMAGE_A
- name: IMAGE_B
deployments:
- name: test
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: A
  patches:
  - path: deployments..image
    op: replace
    value: ${IMAGE_A}
- name: B
  patches:
  - path: deployments..image
    op: replace
    value: ${IMAGE_B}
`,
				options:         &ConfigOptions{Profiles: []string{"B"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"IMAGE_B": "ubuntu"}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "${IMAGE_B}",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profiles with expressions and variables ignored when not activated",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
name: devspace
vars:
- name: IMAGE_A
- name: IMAGE_B
deployments:
- name: test
  helm:
    componentChart: true
    valuesFiles:
      - test.yaml
profiles:
- name: A
  patches:
  - path: deployments..valuesFiles[0]
    op: replace
    value: $(echo ${IMAGE_A})
- name: B
  patches:
  - path: deployments..valuesFiles[0]
    op: replace
    value: $(echo ${IMAGE_B})
`,
				options:         &ConfigOptions{Profiles: []string{"B"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"IMAGE_B": "ubuntu"}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							ValuesFiles: []string{"ubuntu"},
						},
					},
				},
			},
		},
		{
			name: "Profile with name variable",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
vars:
- name: IMAGE_A
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: ${IMAGE_A}
  patches:
  - path: deployments..image
    op: replace
	value: ubuntu
`,
				options:         &ConfigOptions{Profiles: []string{"A"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"IMAGE_A": "production"}},
			},
			expectedErr: "error validating profiles[0]: name cannot be a variable",
		},
		{
			name: "Profile with name expression",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
deployments:
- name: deployment
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: $(echo production)
  patches:
  - path: deployments..image
    op: replace
	value: ubuntu
`,
				options:         &ConfigOptions{Profiles: []string{"production"}},
				generatedConfig: &localcache.LocalCache{},
			},
			expectedErr: "error validating profiles[0]: name cannot be an expression",
		},
		{
			name: "Commands",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
deployments:
- name: ${test_var}
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
commands:
- name: test
  command: should not show up
vars:
- name: unused
  default: unused
profiles:
- name: testprofile
  patches:
  - op: replace
    path: deployments[0].component.containers[0].image
    value: ${unused}
  - op: replace
    path: deployments[0].name
    value: ${should-not-show-up}`,
				options:         &ConfigOptions{Vars: []string{"test_var=test"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Commands: map[string]*latest.CommandConfig{
					"test": &latest.CommandConfig{
						Command: "should not show up",
					},
				},
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Default variables",
			in: &parseTestCaseInput{
				config: `
version: v1beta6
deployments:
- name: ${new}
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
vars:
- name: abc
  default: test123
profiles:
- name: testprofile
  patches:
  - op: replace
    path: vars[0].name
    value: new`,
				options:         &ConfigOptions{Profiles: []string{"testprofile"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"new": "test"}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Variable source none",
			in: &parseTestCaseInput{
				config: `
version: v1beta6
deployments:
- name: ${new}
  kubectl:
    manifests:
    - test.yaml
vars:
- name: new
  source: none
  default: test`,
				options:         &ConfigOptions{},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{
								"test.yaml",
							},
						},
					},
				},
			},
		},
		{
			name: "Profile parent",
			in: &parseTestCaseInput{
				config: `
version: v1beta7
deployments:
- name: test
  kubectl:
    manifests:
    - test.yaml
- name: test2
  kubectl:
    manifests:
    - test.yaml
profiles:
- name: parent
	replace:
		images:
			test:
				image: test
- name: beforeParent
	parent: parent
	patches:
	- op: replace
		path: deployments[0].name
		value: replaced
- name: test
	parent: beforeParent
	patches:
	- op: replace
		path: deployments[0].name
		value: replaced2`,
				options:         &ConfigOptions{Profiles: []string{"test"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars: convertedVars,
				Pipelines: map[string]*latest.Pipeline{
					"build": &latest.Pipeline{
						Run: "build_images test",
					},
					"deploy": &latest.Pipeline{
						Run: `build_images test
create_deployments replaced2 test2 --sequential`,
					},
					"dev": &latest.Pipeline{
						Run: `build_images test
create_deployments replaced2 test2 --sequential

start_dev --all`,
					},
					"purge": &latest.Pipeline{
						Run: `stop_dev --all
purge_deployments replaced2 test2 --sequential`,
					},
				},
				Deployments: map[string]*latest.DeploymentConfig{
					"replaced2": {
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{
								"test.yaml",
							},
						},
					},
					"test2": {
						Kubectl: &latest.KubectlConfig{
							Manifests: []string{
								"test.yaml",
							},
						},
					},
				},
				Images: map[string]*latest.Image{
					"test": {
						Image:           "test",
						RebuildStrategy: latest.RebuildStrategyDefault,
					},
				},
			},
		},
		{
			name: "Inline manifest and normal manifest error",
			in: &parseTestCaseInput{
				config: `
version: v2beta1
name: inline-manifest

deployments:
	test:
		kubectl:
			manifests:
				- test.yaml
			inlineManifest: |-
				kind: Deployment
				apiVersion: apps/v1
				metadata:
					name: test
				spec:
					replicas: 1
					selector:
					matchLabels:
						app.kubernetes.io/component: default
						app.kubernetes.io/name: test
					template:
					metadata:
						labels:
						app.kubernetes.io/component: default
						app.kubernetes.io/name: test
					spec:
						containers:
						- name: default
							image: test
profiles:
- name: test
	replace:
		images:
			test:
				image: test`,
				options:         &ConfigOptions{Profiles: []string{"test"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expectedErr: "deployments[test].kubectl.manifests and deployments[test].kubectl.inlineManifest cannot be used together",
		},

		{
			name: "Profile loop error",
			in: &parseTestCaseInput{
				config: `
version: v1beta7
deployments:
- name: test
- name: test2
profiles:
- name: parent
	parent: test
	replace:
		images:
			test:
				image: test
- name: beforeParent
	parent: parent
	patches:
	- op: replace
		path: deployments[0].name
		value: replaced
- name: test
	parent: beforeParent
	patches:
	- op: replace
		path: deployments[1].name
		value: replaced2`,
				options:         &ConfigOptions{Profiles: []string{"test"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expectedErr: "cannot load config with profile parent: max config loading depth reached. Seems like you have a profile cycle somewhere",
		},
		{
			name: "Port name validation",
			in: &parseTestCaseInput{
				config: `
version: v1beta10
dev:
  ports:
  - name: devbackend
    imageSelector: john/devbackend
    forward:
    - port: 8080
      remotePort: 80
profiles:
- name: production
  patches:
  - op: replace
    path: dev.ports.name=devbackend.imageSelector
    value: john/prodbackend`,
				options:         &ConfigOptions{Profiles: []string{"production"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars: convertedVars,
				Pipelines: map[string]*latest.Pipeline{
					"build":  &latest.Pipeline{},
					"deploy": &latest.Pipeline{},
					"dev": &latest.Pipeline{
						Run: "start_dev --all",
					},
					"purge": &latest.Pipeline{
						Run: "stop_dev --all",
					},
				},
				Dev: map[string]*latest.DevPod{
					"devbackend": {
						ImageSelector: "john/prodbackend",
						Ports: []*latest.PortMapping{
							{
								Port: "8080:80",
							},
						},
					},
				},
			},
		},
		{
			name: "Sync name validation",
			in: &parseTestCaseInput{
				config: `
version: v1beta10
dev:
  sync:
  - name: devbackend
    imageSelector: john/devbackend
    localSubPath: ./
    containerPath: /app
    excludePaths:
    - node_modules/
    - logs/
profiles:
- name: production
  patches:
  - op: replace
    path: dev.sync.name=devbackend.imageSelector
    value: john/prodbackend`,
				options:         &ConfigOptions{Profiles: []string{"production"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars: convertedVars,
				Pipelines: map[string]*latest.Pipeline{
					"build":  &latest.Pipeline{},
					"deploy": &latest.Pipeline{},
					"dev": &latest.Pipeline{
						Run: "start_dev --all",
					},
					"purge": &latest.Pipeline{
						Run: "stop_dev --all",
					},
				},
				Dev: map[string]*latest.DevPod{
					"devbackend": {
						ImageSelector: "john/prodbackend",
						DevContainer: latest.DevContainer{
							Sync: []*latest.SyncConfig{
								{
									Path: "./:/app",
									ExcludePaths: []string{
										"node_modules/",
										"logs/",
									},
								},
							},
							RestartHelper: &latest.RestartHelper{
								Inject: &[]bool{false}[0],
							},
						},
					},
				},
			},
		},
		{
			name: "Patch root path doesn't exist",
			in: &parseTestCaseInput{
				config: `
version: v1beta10
dev:
  sync:
  - name: devbackend
    imageSelector: john/devbackend
    localSubPath: ./
    containerPath: /app
    excludePaths:
    - node_modules/
    - logs/
profiles:
- name: production
  patches:
  - op: add
    path: /images
    value:
      image1:
        image: node
`,
				options:         &ConfigOptions{Profiles: []string{"production"}},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{}},
			},
			expected: &latest.Config{
				Vars: convertedVars,
				Pipelines: map[string]*latest.Pipeline{
					"build": &latest.Pipeline{
						Run: "build_images image1",
					},
					"deploy": &latest.Pipeline{
						Run: `build_images image1`,
					},
					"dev": &latest.Pipeline{
						Run: `build_images image1

start_dev --all`,
					},
					"purge": &latest.Pipeline{
						Run: `stop_dev --all`,
					},
				},
				Images: map[string]*latest.Image{
					"image1": {
						Image:           "node",
						RebuildStrategy: latest.RebuildStrategyDefault,
					},
				},
				Dev: map[string]*latest.DevPod{
					"devbackend": {
						ImageSelector: "john/devbackend",
						DevContainer: latest.DevContainer{
							Sync: []*latest.SyncConfig{
								{
									Path: "./:/app",
									ExcludePaths: []string{
										"node_modules/",
										"logs/",
									},
								},
							},
							RestartHelper: &latest.RestartHelper{Inject: &[]bool{false}[0]},
						},
					},
				},
			},
		},
		{
			name: "Profile activated by matching vars",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
vars:
- name: USE_A
- name: USE_B
deployments:
- name: test
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: A
  activation:
  - vars:
      USE_A: "true"
  patches:
  - path: deployments..image
    op: replace
    value: nginx:a
- name: B
  patches:
  - path: deployments..image
    op: replace
    value: nginx:b
		`,
				options:         &ConfigOptions{},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"USE_A": "true"}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "nginx:a",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Profile not activated by non-matching vars",
			in: &parseTestCaseInput{
				config: `
version: v1beta11
vars:
- name: USE_A
- name: USE_B
deployments:
- name: test
  helm:
    componentChart: true
    values:
      containers:
      - image: nginx
profiles:
- name: A
  activation:
  - vars:
      USE_A: "true"
  patches:
  - path: deployments..image
    op: replace
    value: nginx:a
- name: B
  patches:
  - path: deployments..image
    op: replace
    value: nginx:b
		`,
				options:         &ConfigOptions{},
				generatedConfig: &localcache.LocalCache{Vars: map[string]string{"USE_A": "false"}},
			},
			expected: &latest.Config{
				Vars:      convertedVars,
				Pipelines: testPipeline,
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							Values: map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "nginx",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Expression not resolved",
			in: &parseTestCaseInput{
				config: `
version: v2beta1
deployments:
  test:
    helm:
      releaseName: $$(test)
		`,
				options: &ConfigOptions{},
			},
			expected: &latest.Config{
				Deployments: map[string]*latest.DeploymentConfig{
					"test": {
						Helm: &latest.HelmConfig{
							ReleaseName: "$(test)",
						},
					},
				},
			},
		},
	}

	// Execute test cases
	for _, testCase := range testCases {
		testMap := map[string]interface{}{}
		err := yaml.Unmarshal([]byte(strings.ReplaceAll(testCase.in.config, "	", "  ")), &testMap)
		if err != nil {
			t.Fatalf("Testcase '%s' parsing error: \n%s\n\n %v", testCase.name, strings.ReplaceAll(testCase.in.config, "	", "  "), err)
		}

		cl, err := NewConfigLoader(".")
		if err != nil {
			t.Fatalf("Testcase '%s' failed error: %v", testCase.name, err)
		}

		configLoader := cl.(*configLoader)
		if testCase.in.options == nil {
			testCase.in.options = &ConfigOptions{}
		}
		testCase.in.options.Dry = true
		testMap["name"] = "test"

		newConfig, _, _, err := configLoader.parseConfig(context.Background(),
			testMap,
			testCase.in.generatedConfig,
			&remotecache.RemoteCache{},
			&fakekubectl.Client{},
			NewDefaultParser(),
			testCase.in.options,
			log.Discard)

		if testCase.expectedErr != "" {
			if err == nil {
				t.Fatalf("TestCase %s: expected error, but got none", testCase.name)
			} else if err.Error() != testCase.expectedErr {
				t.Fatalf("TestCase %s: expected error:\n\t%s\nbut got:\n\t%s", testCase.name, testCase.expectedErr, err.Error())
			} else {
				continue
			}
		} else if err != nil {
			t.Fatalf("Error %v in case: '%s'", err, testCase.name)
		}

		testCase.expected.Name = "test"
		testCase.expected.Version = latest.Version
		stripNames(newConfig)
		newConfigYaml, _ := yaml.Marshal(newConfig)
		expectedYaml, _ := yaml.Marshal(testCase.expected)
		assert.Equal(t, string(newConfigYaml), string(expectedYaml), testCase.name)
	}
}

func stripNames(config *latest.Config) {
	for k := range config.Images {
		config.Images[k].Name = ""
	}
	for k := range config.Deployments {
		config.Deployments[k].Name = ""
	}
	for k := range config.Dependencies {
		config.Dependencies[k].Name = ""
	}
	for k := range config.Pipelines {
		config.Pipelines[k].Name = ""
	}
	for k := range config.Dev {
		config.Dev[k].Name = ""
		for c := range config.Dev[k].Containers {
			config.Dev[k].Containers[c].Container = ""
		}
	}
	for k := range config.Vars {
		config.Vars[k].Name = ""
	}
	for k := range config.PullSecrets {
		config.PullSecrets[k].Name = ""
	}
	for k := range config.Commands {
		config.Commands[k].Name = ""
	}
}
