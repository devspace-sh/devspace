package v2

/*type checkDependenciesTestCase struct {
	name string

	dependenciesInChart        []*chart.Chart
	dependenciesInRequirements []*helmchartutil.Dependency

	expectedErr string
}

func TestCheckDependencies(t *testing.T) {
	testCases := []checkDependenciesTestCase{
		checkDependenciesTestCase{
			name:                       "Matching dependencies in chart and requirements",
			dependenciesInChart:        []*chart.Chart{&chart.Chart{Metadata: &chart.Metadata{Name: "MatchingDep"}}},
			dependenciesInRequirements: []*helmchartutil.Dependency{&helmchartutil.Dependency{Name: "MatchingDep"}},
		},
	}

	for _, testCase := range testCases {
		ch := &chart.Chart{
			Dependencies: testCase.dependenciesInChart,
		}
		reqs := &helmchartutil.Requirements{
			Dependencies: testCase.dependenciesInRequirements,
		}

		err := checkDependencies(ch, reqs)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error checking dependencies in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error checking dependencies in testCase %s", testCase.name)
		}
	}
}

type expectedInstallTest struct {
	revision     int32
	chartName    string
	chartVersion string
	values       *chart.Config
}

func TestInstallChart(t *testing.T) {
	installCharts := []*struct {
		releaseName      string
		releaseNamespace string
		values           *map[interface{}]interface{}
		config           *latest.HelmConfig

		expected expectedInstallTest
	}{
		{
			releaseName: "my-release",
			config: &latest.HelmConfig{
				Chart: &latest.ChartConfig{
					Name:    "stable/nginx-ingress",
					Version: "1.24.7",
				},
			},
			expected: expectedInstallTest{
				revision:     1,
				chartName:    "nginx-ingress",
				chartVersion: "1.24.7",
			},
		},
		{
			releaseName: "my-release",
			values: &map[interface{}]interface{}{
				"test": "test",
			},
			config: &latest.HelmConfig{
				Chart: &latest.ChartConfig{
					Name:    "stable/nginx-ingress",
					Version: "1.24.7",
				},
			},
			expected: expectedInstallTest{
				revision: 2,
				values: &chart.Config{
					Raw: "test: test\n",
				},
			},
		},
		{
			releaseName:      "my-release-2",
			releaseNamespace: "other-namespace",
			config: &latest.HelmConfig{
				Chart: &latest.ChartConfig{
					Name:    "stable/nginx-ingress",
					Version: "1.24.7",
				},
			},
			expected: expectedInstallTest{
				revision:     1,
				chartName:    "nginx-ingress",
				chartVersion: "1.24.7",
			},
		},
	}

	config := createFakeConfig()

	// Create the fake client.
	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}
	helmClient := &helm.FakeClient{}

	client, err := create(config, configutil.TestNamespace, helmClient, kubeClient, false, log.GetInstance())
	if err != nil {
		t.Fatal(err)
	}

	err = client.UpdateRepos(log.GetInstance())
	if err != nil {
		t.Fatal(err)
	}

	for _, i := range installCharts {
		installResponse, err := client.InstallChart(i.releaseName, i.releaseNamespace, i.values, i.config)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, installResponse.GetName(), i.releaseName)
		if i.releaseNamespace == "" {
			assert.Equal(t, installResponse.GetNamespace(), "default")
		} else {
			assert.Equal(t, installResponse.GetNamespace(), i.releaseNamespace)
		}
		assert.Equal(t, installResponse.GetVersion(), i.expected.revision)
		assert.Equal(t, installResponse.GetChart().GetMetadata().GetName(), i.expected.chartName)
		assert.Equal(t, installResponse.GetChart().GetMetadata().GetVersion(), i.expected.chartVersion)
		if i.expected.values != nil {
			assert.DeepEqual(t, installResponse.GetConfig(), i.expected.values)
		}
	}
}

type analyzeErrorTestCase struct {
	name string

	inputErr    error
	namespace   string
	createdPods []*k8sv1.Pod

	expectedErr string
}

func TestAnalyzeError(t *testing.T) {
	testCases := []analyzeErrorTestCase{
		analyzeErrorTestCase{
			name:        "Test analyze no-timeout error",
			inputErr:    errors.Errorf("Some error"),
			expectedErr: "Some error",
		},
		analyzeErrorTestCase{
			name:      "Test analyze timeout error",
			inputErr:  errors.Errorf("timed out waiting"),
			namespace: "testNS",
		},
	}

	for _, testCase := range testCases {
		config := createFakeConfig()

		// Create the fake client.
		kubeClient := &kubectl.Client{
			Client: fake.NewSimpleClientset(),
		}
		helmClient := &helm.FakeClient{}

		for _, pod := range testCase.createdPods {
			_, err := kubeClient.Client.CoreV1().Pods(testCase.namespace).Create(pod)
			assert.NilError(t, err, "Error creating testPod in testCase %s", testCase.name)
		}

		client, err := create(config, configutil.TestNamespace, helmClient, kubeClient, false, &log.DiscardLogger{})
		if err != nil {
			t.Fatal(err)
		}

		err = client.analyzeError(testCase.inputErr, testCase.namespace)
		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error analyzing error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error returned in testCase %s", testCase.name)
		}
	}
}*/
