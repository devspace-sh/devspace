package loader

import (
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"gotest.tools/assert"
)

func TestValidateImageName(t *testing.T) {
	config := &latest.Config{
		Images: map[string]*latest.ImageConfig{
			"default": {
				Image: "localhost:5000/node",
			},
		},
	}
	err := validateImages(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Images: map[string]*latest.ImageConfig{
			"default": {
				Image: "localhost:5000/node:latest",
			},
		},
	}
	err = validateImages(config)
	assert.Error(t, err, "images.default.image 'localhost:5000/node:latest' can not have tag 'latest'")
}

func TestValidateHooks(t *testing.T) {
	config := &latest.Config{
		Hooks: []*latest.HookConfig{
			{
				Events: []string{
					"after:deploy:my-deployment",
				},
				Container: &latest.HookContainer{
					ContainerName: "fakeContainer",
					LabelSelector: map[string]string{
						"app": "selectThisContainer",
					},
				},
				Command: "doSomething",
			},
		},
	}

	err := validateHooks(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Hooks: []*latest.HookConfig{
			{
				Events: []string{
					"after:deploy:my-deployment",
				},
				Container: &latest.HookContainer{
					ContainerName: "fakeContainer",
				},
				Command: "doSomething",
			},
		},
	}

	err = validateHooks(config)
	assert.Error(t, err, "hooks[0].container.containerName is defined but hooks[0].container.labelSelector is not defined")
}

func TestValidateDev(t *testing.T) {
	// test port forwarding
	localPort := int(8080)
	remotePort := int(9090)
	config := &latest.Config{
		Dev: latest.DevConfig{
			Ports: []*latest.PortForwardingConfig{
				{
					ContainerName: "fakeContainer",
					Name:          "someName",
					ImageSelector: "selecMe",
					LabelSelector: map[string]string{
						"app": "MeApp",
					},
					PortMappings: []*latest.PortMapping{
						{
							LocalPort:  &localPort,
							RemotePort: &remotePort,
						},
					},
				},
			},
		},
	}

	err := validateDev(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Dev: latest.DevConfig{
			Ports: []*latest.PortForwardingConfig{
				{
					ContainerName: "fakeContainer",
					Name:          "someName",
					PortMappings: []*latest.PortMapping{
						{
							LocalPort:  &localPort,
							RemotePort: &remotePort,
						},
					},
				},
			},
		},
	}

	err = validateDev(config)
	assert.Error(t, err, "Error in config: containerName is defined but label selector is nil in ports config at index 0")

	// test sync
	config = &latest.Config{
		Dev: latest.DevConfig{
			Sync: []*latest.SyncConfig{
				{
					ContainerName: "fakeContainer",
					Name:          "someName",
					ImageSelector: "selecMe",
					LabelSelector: map[string]string{
						"app": "MeApp",
					},
				},
			},
		},
	}

	err = validateDev(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Dev: latest.DevConfig{
			Sync: []*latest.SyncConfig{
				{
					ContainerName: "fakeContainer",
					Name:          "someName",
				},
			},
		},
	}

	err = validateDev(config)
	assert.Error(t, err, "Error in config: containerName is defined but label selector is nil in sync config at index 0")

	// test replace pods
	config = &latest.Config{
		Dev: latest.DevConfig{
			ReplacePods: []*latest.ReplacePod{
				{
					ContainerName: "fakeContainer",
					LabelSelector: map[string]string{
						"app": "MeApp",
					},
				},
			},
		},
	}

	err = validateDev(config)
	assert.NilError(t, err)

	config = &latest.Config{
		Dev: latest.DevConfig{
			ReplacePods: []*latest.ReplacePod{
				{
					ContainerName: "fakeContainer",
				},
			},
		},
	}

	err = validateDev(config)
	assert.Error(t, err, "Error in config: containerName is defined but label selector is nil in replace pods at index 0")
}
