package v2

import ()

/*func createFakeConfig() *latest.Config {
	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name:      "test-deployment",
				Namespace: loader.TestNamespace,
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: "stable/nginx",
					},
				},
			},
			&latest.DeploymentConfig{
				Name: "test-deployment",
				Helm: &latest.HelmConfig{
					Chart: &latest.ChartConfig{
						Name: "stable/nginx",
					},
				},
			},
		},
	}

	loader.SetFakeConfig(testConfig)
	return testConfig
}

func TestCreateTiller(t *testing.T) {
	config := createFakeConfig()

	// Create the fake client.
	client := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	err := createTillerRBAC(config, client, "tiller-namespace", log.GetInstance())
	if err != nil {
		t.Fatal(err)
	}
}*/
