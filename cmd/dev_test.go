package cmd

type devTestCase struct{}

/*func TestDev(t *testing.T) {
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

testCases := []devTestCase{
	devTestCase{
		name:           "Invalid flags",
		fakeConfig:     &latest.Config{},
		skipBuildFlag:  true,
		forceBuildFlag: true,
		expectedErr:    "Flags --skip-build & --force-build cannot be used together",
	},
	/*devTestCase{
		name:       "interactive without images",
		fakeConfig: &latest.Config{},
		fakeKubeClient: &kubectl.Client{
			Client:         fake.NewSimpleClientset(),
			CurrentContext: "minikube",
		},
		files: map[string]interface{}{
			constants.DefaultConfigPath: &latest.Config{
				Version: latest.Version,
			},
		},
		interactiveFlag: true,
		expectedErr:     "Your configuration does not contain any images to build for interactive mode. If you simply want to start the terminal instead of streaming the logs, run `devspace dev -t`",
	},
	devTestCase{
		name:       "Cloud Space can't be resumed",
		fakeConfig: &latest.Config{},
		fakeKubeClient: &kubectl.Client{
			Client:         fake.NewSimpleClientset(),
			CurrentContext: "minikube",
		},
		files: map[string]interface{}{
			constants.DefaultConfigPath: &latest.Config{
				Version: latest.Version,
				Dev: &latest.DevConfig{
					Interactive: &latest.InteractiveConfig{},
				},
			},
		},
		expectedErr:    "is cloud space: Unable to get AuthInfo for kube-context: Unable to find kube-context 'minikube' in kube-config file",
	},*/ /*
	}

	log.OverrideRuntimeErrorHandler(true)
	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testDev(t, testCase)
	}
}

func testDev(t *testing.T, testCase devTestCase) {
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
	loader.ResetConfig()
	generated.ResetConfig()
	kubeconfig.SetFakeConfig(testCase.fakeKubeConfig)
	kubectl.SetFakeClient(testCase.fakeKubeClient)

	for path, content := range testCase.files {
		asYAML, err := yaml.Marshal(content)
		assert.NilError(t, err, "Error parsing config to yaml in testCase %s", testCase.name)
		err = fsutil.WriteToFile(asYAML, path)
		assert.NilError(t, err, "Error writing file in testCase %s", testCase.name)
	}

	err = (&DevCmd{
		GlobalFlags: &testCase.globalFlags,

		AllowCyclicDependencies: testCase.allowCyclicDependenciesFlag,
		SkipPush:                testCase.skipPushFlag,

		ForceBuild:        testCase.forceBuildFlag,
		SkipBuild:         testCase.skipBuildFlag,
		BuildSequential:   testCase.buildSequentialFlag,
		ForceDeploy:       testCase.forceDeploymentFlag,
		Deployments:       testCase.deploymentsFlag,
		ForceDependencies: testCase.forceDependenciesFlag,

		Sync:            testCase.syncFlag,
		Terminal:        testCase.terminalFlag,
		ExitAfterDeploy: testCase.exitAfterDeployFlag,
		SkipPipeline:    testCase.skipPipelineFlag,
		Portforwarding:  testCase.portForwardingFlag,
		VerboseSync:     testCase.verboseSyncFlag,
		Interactive:     testCase.interactiveFlag,
	}).Run(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)
	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}*/
