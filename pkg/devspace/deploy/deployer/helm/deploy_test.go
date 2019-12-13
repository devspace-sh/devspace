package helm

import ()

// Test namespace to create
const testNamespace = "test-helm-deploy"

/*func TestHelmDeployment(t *testing.T) {
	namespace := "testnamespace"
	chartName := "chart"

	// 1. Create fake config & generated config
	deployConfig := &latest.DeploymentConfig{
		Name: "test-deployment",
		Helm: &latest.HelmConfig{
			TillerNamespace: namespace,
			Chart: &latest.ChartConfig{
				Name: chartName,
			},
			ValuesFiles: []string{"chart"},
		},
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: "node",
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

	// 4. Deploy test chart
	helm, err := New(testConfig, kubeClient, deployConfig, log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating helm client: %v", err)
	}
	helm.Helm = otherhelmpackage.NewFakeClient(kubeClient.Client, namespace)
	isDeployed, err := helm.Deploy(generatedConfig.Profiles["default"], true, nil)
	if err != nil {
		t.Fatalf("Error deploying chart: %v", err)
	}

	// 5. Validate deployed chart & test .Status function
	assert.Equal(t, true, isDeployed)

	status, err := helm.Status()
	if err != nil {
		t.Fatalf("Error checking status: %v", err)
	}
	if strings.HasPrefix(status.Status, "Deployed") == false {
		t.Fatalf("Unexpected deployment status: %s != Deployed", status.Status)
	}

	// 6. Delete test chart
	err = helm.Delete(generatedConfig.Profiles["default"])
	if err != nil {
		t.Fatalf("Error deleting chart: %v", err)
	}

	// 7. Delete test namespace
	err = kubeClient.Client.CoreV1().Namespaces().Delete(namespace, nil)
	if err != nil {
		t.Fatalf("Error deleting namespace: %v", err)
	}
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

	return nil
}

func TestReplaceImageNames(t *testing.T) {
	config := map[string]*latest.ImageConfig{
		"test-2": &latest.ImageConfig{
			Image: "simple-replace",
		},
		"test-3": &latest.ImageConfig{
			Image: "prefix/simple-replace",
		},
		"test-4": &latest.ImageConfig{
			Image: "test.com/prefix/simple-replace",
		},
		"test-5": &latest.ImageConfig{
			Image: "test.com:5000/prefix/simple-replace",
		},
	}
	cache := &generated.CacheConfig{
		Images: map[string]*generated.ImageCache{
			"test-1": &generated.ImageCache{
				ImageName: "dont-replace-me",
				Tag:       "",
			},
			"test-2": &generated.ImageCache{
				ImageName: "simple-replace",
				Tag:       "replaced",
			},
			"test-3": &generated.ImageCache{
				ImageName: "prefix/simple-replace",
				Tag:       "replaced",
			},
			"test-4": &generated.ImageCache{
				ImageName: "test.com/prefix/simple-replace",
				Tag:       "replaced",
			},
			"test-5": &generated.ImageCache{
				ImageName: "test.com:5000/prefix/simple-replace",
				Tag:       "replaced",
			},
		},
	}
	builtImages := map[string]string{
		"simple-replace": "",
	}

	input := map[interface{}]interface{}{
		"imagename": "dont-replace-me",
		"simple-replace": []interface{}{
			map[interface{}]interface{}{
				"replace1": "simple-replace",
				"replace2": "  simple-replace:tag ",
				"test":     "ssimple-replace",
				"other": map[interface{}]interface{}{
					"replace1": "prefix/simple-replace",
					"replace2": "test.com/prefix/simple-replace",
					"replace3": "test.com:5000/prefix/simple-replace:latest",
				},
			},
		},
	}
	output := map[interface{}]interface{}{
		"imagename": "dont-replace-me",
		"simple-replace": []interface{}{
			map[interface{}]interface{}{
				"replace1": "simple-replace:replaced",
				"replace2": "simple-replace:replaced",
				"test":     "ssimple-replace",
				"other": map[interface{}]interface{}{
					"replace1": "prefix/simple-replace:replaced",
					"replace2": "test.com/prefix/simple-replace:replaced",
					"replace3": "test.com:5000/prefix/simple-replace:replaced",
				},
			},
		},
	}

	shouldRedeploy := replaceContainerNames(input, cache, config, builtImages)
	if shouldRedeploy == false {
		t.Fatal("Expected to redeploy")
	}

	isEqual := reflect.DeepEqual(input, output)
	if !isEqual {
		gotYaml, _ := yaml.Marshal(input)
		expectedYaml, _ := yaml.Marshal(output)

		t.Fatalf("Replace failed: Got\n %s\n, but expected\n %s", gotYaml, expectedYaml)
	}

	shouldRedeploy = replaceContainerNames(input, cache, config, builtImages)
	if shouldRedeploy == false {
		t.Fatal("Expected no redeploy")
	}

	isEqual = reflect.DeepEqual(input, output)
	if !isEqual {
		gotYaml, _ := yaml.Marshal(input)
		expectedYaml, _ := yaml.Marshal(output)

		t.Fatalf("Replace failed: Got\n %s\n, but expected\n %s", gotYaml, expectedYaml)
	}
}*/
