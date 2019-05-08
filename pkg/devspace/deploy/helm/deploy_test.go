package helm

import "testing"

// Test namespace to create
const testNamespace = "test-helm-deploy"

func TestHelmDeployment(t *testing.T) {
	// @Florian

	// 1. Create fake config & generated config
	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: &[]*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: &map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: ptr.String("nginx"),
			},
		},
	}
	configutil.SetFakeConfig(testConfig)

	// Create fake generated config
	generatedConfig := &generated.Config{
		ActiveConfig: "default",
		Configs: map[string]*generated.CacheConfig{
			"default": &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"default": &generated.ImageCache{
						Tag: "1.15", // This will be appended to nginx during deploy
					},
				},
			},
		},
	}
	generated.InitDevSpaceConfig(generatedConfig, "default")

	// 2. Write test chart into a temp folder
	dir, err := ioutil.TempDir("", "testDeploy")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}

	copy.Copy("./../../../../examples/kaniko", dir)

	wdBackup, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	err = os.Chdir(dir)
	if err != nil {
		t.Fatalf("Error changing working directory: %v", err)
	}

	// 8. Delete temp folder
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	// 3. Init kubectl & create test namespace
	// 4. Deploy test chart
	// 5. Validate deployed chart & test .Status function
	// 6. Delete test chart
	// 7. Delete test namespace
}
