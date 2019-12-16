package cmd

import ()

/*type deployTestCase struct {
	name string

	fakeConfig       *latest.Config
	fakeKubeConfig   clientcmd.ClientConfig
	fakeKubeClient   kubectl.Client
	files            map[string]interface{}
	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider

	forceBuildFlag              bool
	skipBuildFlag               bool
	buildSequentialFlag         bool
	forceDeployFlag             bool
	deploymentsFlag             string
	forceDependenciesFlag       bool
	skipPushFlag                bool
	allowCyclicDependenciesFlag bool
	globalFlags                 flags.GlobalFlags

	expectedErr string
}

func TestDeploy(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

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
		err = os.RemoveAll(dir)
		if err != nil {
			t.Fatalf("Error removing dir: %v", err)
		}
	}()

	testCases := []deployTestCase{
		deployTestCase{
			name:           "Invalid flags",
			fakeConfig:     &latest.Config{},
			skipBuildFlag:  true,
			forceBuildFlag: true,
			expectedErr:    "Flags --skip-build & --force-build cannot be used together",
		},
		deployTestCase{
			name:       "Successfully deployed nothing",
			fakeConfig: &latest.Config{},
			fakeKubeClient: &kubectl.Client{
				Client:         fake.NewSimpleClientset(),
				CurrentContext: "minikube",
			},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"minikube": &clientcmdapi.Context{},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"": &clientcmdapi.AuthInfo{},
					},
				},
			},
			deploymentsFlag: " ",
		},
	}

	log.OverrideRuntimeErrorHandler(true)
	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testDeploy(t, testCase)
	}
}

func testDeploy(t *testing.T, testCase deployTestCase) {
	defer func() {
		for path := range testCase.files {
			removeTask := strings.Split(path, "/")[0]
			err := os.RemoveAll(removeTask)
			assert.NilError(t, err, "Error cleaning up folder in testCase %s", testCase.name)
		}
		err := os.RemoveAll(log.Logdir)
		assert.NilError(t, err, "Error cleaning up folder in testCase %s", testCase.name)
	}()

	cloudpkg.DefaultGraphqlClient = &customGraphqlClient{
		responses: testCase.graphQLResponses,
	}

	providerConfig, err := cloudconfig.Load()
	assert.NilError(t, err, "Error getting provider config in testCase %s", testCase.name)
	providerConfig.Providers = testCase.providerList

	loader.SetFakeConfig(testCase.fakeConfig)
	generated.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)
	kubectl.SetFakeClient(testCase.fakeKubeClient)

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err = (&DeployCmd{
		GlobalFlags: &testCase.globalFlags,

		ForceBuild:        testCase.forceBuildFlag,
		SkipBuild:         testCase.skipBuildFlag,
		BuildSequential:   testCase.buildSequentialFlag,
		ForceDeploy:       testCase.forceDeployFlag,
		Deployments:       testCase.deploymentsFlag,
		ForceDependencies: testCase.forceDependenciesFlag,

		SkipPush:                testCase.skipPushFlag,
		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}

	err = filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		os.RemoveAll(path)
		return nil
	})
	assert.NilError(t, err, "Error cleaning up in testCase %s", testCase.name)
}*/
