package helper

import ()

/*var expectedAbsoluteContextPath, expectedAbsoluteDockerfilePath string
var expectedEntryPoint *[]*string
var expectedLog log.Logger
var usedT *testing.T
var buildImageCalled = false
var returnErr error

type fakeBuilder struct{}

func (builder fakeBuilder) BuildImage(absoluteContextPath string, absoluteDockerfilePath string, entrypoint []string, cmd []string, log log.Logger) error {
	assert.Equal(usedT, expectedAbsoluteContextPath, absoluteContextPath, "Wrong context path given to builder")
	assert.Equal(usedT, expectedAbsoluteDockerfilePath, absoluteDockerfilePath, "Wrong dockerfile path given to builder")
	assert.Equal(usedT, expectedEntryPoint, expectedEntryPoint, "Wrong entryPoints given to builder")
	assert.Equal(usedT, expectedLog, log, "Wrong logger given to builder")
	buildImageCalled = true
	return returnErr
}

func TestBuild(t *testing.T) {
	testConfig := &latest.Config{}
	imageConfig := &latest.ImageConfig{
		Image:      "SomeImage",
		Dockerfile: "Dockerfile",
		Context:    "ImageConfigContext",
	}
	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}
	helper := NewBuildHelper(testConfig, kubeClient, "engineName", "imageConfigName", imageConfig, "imageTag", true)

	var err error
	expectedAbsoluteContextPath, err = filepath.Abs("ImageConfigContext")
	assert.NilError(t, err, "Error getting absolute path")
	expectedAbsoluteDockerfilePath, err = filepath.Abs("Dockerfile")
	assert.NilError(t, err, "Error getting absolute path")
	expectedLog = &log.DiscardLogger{}
	usedT = t
	returnErr = nil

	err = helper.Build(fakeBuilder{}, expectedLog)
	assert.NilError(t, err, "Error building image")
	assert.Equal(t, true, buildImageCalled, "BuildImage of ImageBuilder is not called")

	returnErr = errors.Errorf("SomeErr")
	buildImageCalled = false
	err = helper.Build(fakeBuilder{}, expectedLog)
	assert.Equal(t, true, buildImageCalled, "BuildImage of ImageBuilder is not called")
	assert.Error(t, err, "Error during image build: SomeErr", "No or wrong error passed")
}

func TestShouldRebuild(t *testing.T) {
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

	helper := &BuildHelper{
		DockerfilePath:  "Doesn'tExist",
		ImageConf:       &latest.ImageConfig{},
		Entrypoint:      []string{"echo"},
		ImageConfigName: "ImageConf",
	}
	cache := &generated.CacheConfig{
		Images: map[string]*generated.ImageCache{},
	}

	expectedErrorString := "Dockerfile Doesn'tExist missing: CreateFile Doesn'tExist: The system cannot find the file specified."
	if runtime.GOOS != "windows" {
		expectedErrorString = "Dockerfile Doesn'tExist missing: stat Doesn'tExist: no such file or directory"
	}
	shouldRebuild, err := helper.ShouldRebuild(cache, false)
	assert.Error(t, err, expectedErrorString)
	assert.Equal(t, false, shouldRebuild, "After an error occurred a rebuild is recommended.")

	helper.DockerfilePath = "IsFile"
	err = fsutil.WriteToFile([]byte(""), "IsFile")
	assert.NilError(t, err, "Error creating File")
	shouldRebuild, err = helper.ShouldRebuild(cache, false)
	assert.NilError(t, err, "Error when asking whether we should rebuild with basic setting")
	assert.Equal(t, true, shouldRebuild, "After an error occurred a rebuild is recommended.")
	assert.Equal(t, false, cache.Images["ImageConf"].DockerfileHash == "", "DockerfileHash not set")
	assert.Equal(t, false, cache.Images["ImageConf"].ContextHash == "", "ContextHash not set")
	assert.Equal(t, false, cache.Images["ImageConf"].ImageConfigHash == "", "ImageConfigHash not set")
	assert.Equal(t, false, cache.Images["ImageConf"].EntrypointHash == "", "EntrypointHash not set")
}*/
