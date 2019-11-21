package kubectl

import (
	"os"
	"testing"
)

// Test namespace to create
const testNamespace = "test-kubectl-deploy"

// Test namespace to create
const testKustomizeNamespace = "test-kubectl-kustomize-deploy"

// @MoreTests
//When kubectl is testable, test it

/*func TestKubectlManifests(t *testing.T) {
	t.Skip("Not yet testable")
	namespace := "testnamespace"
	// 1. Create fake config & generated config

	// Create fake devspace config
	deploymentConfig := &latest.DeploymentConfig{
		Name: "test-deployment",
		Kubectl: &latest.KubectlConfig{
			Manifests: []string{"kube"},
			Flags:     []string{"--dry-run"},
		},
	}
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			deploymentConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: "nginx",
			},
		},
	}
	loader.SetFakeConfig(testConfig)

	// Create fake generated config
	generatedConfig := &generated.Config{
		ActiveProfile: "default",
		Profiles: map[string]*generated.CacheConfig{
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

	// 2. Write test manifests into a temp folder
	dir, err := ioutil.TempDir("", "testFolder")
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

	// 8. Delete temp folder
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

	err = makeTestProject(dir)
	if err != nil {
		t.Fatalf("Error creating test project: %v", err)
	}

	// 3. Init kubectl & create test namespace
	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}
	_, err = kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	// 4. Deploy manifests
	deployConfig, err := New(testConfig, kubeClient, deploymentConfig, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating deployConfig: %v", err)
	}

	isDeployed, err := deployConfig.Deploy(generatedConfig.Profiles["default"], true, nil)
	if err != nil {
		t.Fatalf("Error deploying chart: %v", err)
	}
	assert.Equal(t, true, isDeployed, "Manifest is not deployed. No errors returned.")
	// 5. Validate manifests
	// 6. Delete manifests
	// 7. Delete test namespace
}*/

func TestKubectlManifestsWithKustomize(t *testing.T) {
	// @MoreTests
	// 1. Create fake config & generated config
	// 2. Write test kustomize files (see examples) into a temp folder
	// 3. Init kubectl & create test namespace
	// 4. Deploy files
	// 5. Validate deployed resources
	// 6. Delete deployed files
	// 7. Delete test namespace
	// 8. Delete temp folder
}

func makeTestProject(dir string) error {
	file, err := os.Create("package.json")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`{
  "name": "node-js-sample",
  "version": "0.0.1",
  "description": "A sample Node.js app using Express 4",
  "main": "index.js",
  "scripts": {
    "start": "nodemon index.js"
  },
  "dependencies": {
    "express": "^4.13.3",
    "nodemon": "^1.18.4",
    "request": "^2.88.0"
  },
  "keywords": [
    "node",
    "express"
  ],
  "license": "MIT"
}`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	file, err = os.Create("index.js")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`var express = require('express');
var request = require('request');
var app = express();

app.get('/', async (req, res) => {
  var body = await new Promise((resolve, reject) => {
    request('http://php/index.php', (err, res, body) => {
      if (err) { 
        reject(err);
        return;
      }

      resolve(body);
    });
  });

  res.send(body);
});

app.listen(3000, function () {
  console.log('Example app listening on port 3000!');
});`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	file, err = os.Create("Dockerfile")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`FROM node:8.11.4

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . .

CMD ["npm", "start"]`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	file, err = os.Create(".dockerignore")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`Dockerfile
.devspace/
chart/
node_modules/`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	fileInfo, err := os.Lstat(".")
	if err != nil {
		return err
	}
	err = os.Mkdir("kube", fileInfo.Mode())
	if err != nil {
		return err
	}

	file, err = os.Create("kube/deployment.yaml")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: devspace
spec:
  replicas: 1
  selector:
    matchLabels:
      release: devspace-node
  template:
    metadata:
      labels:
        release: devspace-node
    spec:
      containers:
      - name: node
        image: node`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	return nil
}
