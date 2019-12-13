package build

import ()

/*func TestBuild(t *testing.T) {
	t.Skip("Not yet testable because docker client must be faked")

	//Create tempDir and go into it
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

	// Delete temp folder after test
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
		},
	}
	loader.SetFakeConfig(testConfig)

	cache := &generated.CacheConfig{
		Images: make(map[string]*generated.ImageCache) ,
	}
	kubeClient := fake.NewSimpleClientset()

	//Test without images
	go makeAllPodsRunning(t, kubeClient, loader.TestNamespace)
	images, err := All(testConfig, cache, &kubectl.Client{Client: kubeClient}, true, true, true, true, true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error building all 0 images: %v", err)
	}
	assert.Equal(t, 0, len(images), "Images returned without any image declared in config")

	//Test with one image
	testConfig.Images = map[string]*latest.ImageConfig{}
	testConfig.Images["firstimg"] = &latest.ImageConfig{
		Image: "firstimg",
	}
	images, err = All(testConfig, cache, &kubectl.Client{Client: kubeClient}, true, true, true, false, true, log.GetInstance())
	if err != nil {
		t.Fatalf("Error building all 1 images: %v", err)
	}
	assert.Equal(t, false, testConfig.Images["firstimg"] == nil, "Images returned without any image declared in config")
}

func makeAllPodsRunning(t *testing.T, kubeClient *fake.Clientset, namespace string) {
	time.Sleep(time.Second)

	podList, err := kubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error listing pods of fake kubeClient: %v", err)
	}
	for _, pod := range podList.Items {
		pod.Status.InitContainerStatuses = []v1.ContainerStatus{
		  v1.ContainerStatus{
				State: v1.ContainerState{
					Running: &v1.ContainerStateRunning{},
				},
			},
		}
		kubeClient.CoreV1().Pods(namespace).Update(&pod)
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

	return nil
}*/
