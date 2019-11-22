package deploy

import (
)

// Test namespace to create
/*const testNamespace = "test-helm-deploy"

func TestHelmDeployment(t *testing.T) {
	namespace := "testnamespace"
	valuesFiles := make([]*string, 1)
	valuesFiles0 := "chart"
	valuesFiles[0] = &valuesFiles0

	// 1. Create fake config & generated config

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: "don'tDeploy",
			},
			&latest.DeploymentConfig{
				Name: "test-deployment",
				Kubectl: &latest.KubectlConfig{
					Manifests: []string{},
				},
			},
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

	// 2. Write test chart into a temp folder
	dir, err := ioutil.TempDir("", "testDeploy")
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

	cache := &generated.CacheConfig{
		Deployments: make(map[string]*generated.DeploymentCache),
	}

	// 4. Deploy
	err = All(testConfig, cache, kubeClient, true, true, map[string]string{"default": "nginx"}, []string{"test-deployment"}, &log.DiscardLogger{})
	if err != nil {
		t.Fatalf("Error deploying all: %v", err)
	}

	testConfig.Deployments = []*latest.DeploymentConfig{
		&latest.DeploymentConfig{
			Name:    "test-deployment",
			Kubectl: &latest.KubectlConfig{},
		},
	}
	err = All(testConfig, cache, kubeClient, true, true, map[string]string{"default": "nginx"}, []string{"test-deployment"}, &log.DiscardLogger{})
	if err == nil {
		t.Fatal("No Error deploying with an invalid Kubectl in deployment config.")
	}

	testConfig.Deployments = []*latest.DeploymentConfig{
		&latest.DeploymentConfig{
			Name: "test-deployment",
		},
	}
	err = All(testConfig, cache, kubeClient, true, true, map[string]string{"default": "nginx"}, []string{"test-deployment"}, &log.DiscardLogger{})
	if err == nil {
		t.Fatal("No Error deploying with no deployClient in deployment conig.")
	}

	// 7. Delete test namespace
	err = kubeClient.Client.CoreV1().Namespaces().Delete(namespace, nil)
	if err != nil {
		t.Fatalf("Error deleting namespace: %v", err)
	}
}

func TestPurgeDeployments(t *testing.T) {
	namespace := "testnamespace"
	valuesFiles := make([]*string, 1)
	valuesFiles0 := "chart"
	valuesFiles[0] = &valuesFiles0

	// 1. Create fake config & generated config

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			&latest.DeploymentConfig{
				Name: "test-deployment",
				Kubectl: &latest.KubectlConfig{
					Manifests: []string{},
				},
			},
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

	// 2. Write test chart into a temp folder
	dir, err := ioutil.TempDir("", "testDeploy")
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

	cache := &generated.CacheConfig{
		Deployments: make(map[string]*generated.DeploymentCache),
	}
	PurgeDeployments(testConfig, cache, kubeClient, []string{}, &log.DiscardLogger{})
	testConfig.Deployments = []*latest.DeploymentConfig{
		&latest.DeploymentConfig{
			Name: "test-deployment",
			Kubectl: &latest.KubectlConfig{
				Manifests: []string{},
			},
		},
		&latest.DeploymentConfig{
			Name: "NotListed",
		},
	}
	PurgeDeployments(testConfig, cache, kubeClient, []string{"test-deployment"}, &log.DiscardLogger{})

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
	_, err = file.Write([]byte(`apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: devspace
spec:
  replicas: 1
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
	err = os.Mkdir("chart", fileInfo.Mode())
	if err != nil {
		return err
	}

	file, err = os.Create("chart/Chart.yaml")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`name: my-app
version: v0.0.2
description: A Kubernetes-Native Application`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	file, err = os.Create("chart/values.yaml")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`image: devspace`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	err = os.Mkdir("chart/templates", fileInfo.Mode())
	if err != nil {
		return err
	}

	file, err = os.Create("chart/templates/deployment.yaml")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`kind: Deployment
apiVersion: extensions/v1beta1
metadata:
  name: devspace
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/component: {{ $.Release.Name | quote }}
        app.kubernetes.io/name: devspace-app
        helm.sh/chart: "{{ $.Chart.Name }}-{{ $.Chart.Version }}"
    spec:
      containers:
        - name: default
          image: {{ .Values.image }}`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	file, err = os.Create("chart/templates/service.yaml")
	if err != nil {
		return err
	}
	_, err = file.Write([]byte(`apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: devspace-app
  name: external
spec:
  ports:
  - name: port-0
    port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    app.kubernetes.io/component: {{ $.Release.Name | quote }}
    app.kubernetes.io/name: devspace-app
  type: ClusterIP`))
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	err = fsutil.WriteToFile([]byte(""), "TestManifest.yaml")
	if err != nil {
		return err
	}

	return nil
}*/
