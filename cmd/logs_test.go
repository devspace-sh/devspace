package cmd

/*type logsTestCase struct {
	name string

	fakeConfig       *latest.Config
	fakeKubeConfig   clientcmd.ClientConfig
	fakeKubeClient   kubectl.Client
	files            map[string]interface{}
	graphQLResponses []interface{}
	providerList     []*cloudlatest.Provider
	answers          []string

	labelSelectorFlag     string
	containerFlag         string
	podFlag               string
	pickFlag              bool
	followFlag            bool
	lastAmountOfLinesFlag int
	globalFlags           flags.GlobalFlags

	expectedErr string
}

func TestLogs(t *testing.T) {
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
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
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

	testCases := []logsTestCase{
		logsTestCase{
			name:       "No resources",
			fakeConfig: &latest.Config{},
			fakeKubeClient: &kubectl.Client{
				Client: fake.NewSimpleClientset(),
			},
			fakeKubeConfig: &customKubeConfig{
				rawconfig: clientcmdapi.Config{
					Contexts: map[string]*clientcmdapi.Context{
						"": &clientcmdapi.Context{},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"": &clientcmdapi.AuthInfo{},
					},
				},
			},
			pickFlag:    true,
			expectedErr: "Couldn't find a running pod in namespace ",
		},
	}

	log.SetInstance(&log.DiscardLogger{PanicOnExit: true})

	for _, testCase := range testCases {
		testLogs(t, testCase)
	}
}

func testLogs(t *testing.T, testCase logsTestCase) {
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

	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
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

	err = (&LogsCmd{
		LabelSelector:     testCase.labelSelectorFlag,
		Container:         testCase.containerFlag,
		Pod:               testCase.podFlag,
		Pick:              testCase.pickFlag,
		Follow:            testCase.followFlag,
		LastAmountOfLines: testCase.lastAmountOfLinesFlag,
		GlobalFlags:       &testCase.globalFlags,
	}).RunLogs(nil, []string{})

	if testCase.expectedErr == "" {
		assert.NilError(t, err, "Unexpected error in testCase %s.", testCase.name)

	} else {
		assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s.", testCase.name)
	}
}
*/
