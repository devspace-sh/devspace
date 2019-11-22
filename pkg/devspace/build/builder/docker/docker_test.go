package docker

import ()

//@Moretest
//Coverage is 46% and that's not enough

/*func TestDockerBuild(t *testing.T) {
	t.Skip("For some reason there's a problem with the coverage in this package")

	// 1. Write test dockerfile and context to a temp folder
	dir, err := ioutil.TempDir("", "testDocker")
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

	// 4. Cleanup temp folder
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

	deployConfig := &latest.DeploymentConfig{
		Name: "test-deployment",
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: "nginx",
			},
		},
	}
	loader.SetFakeConfig(testConfig)

	dockerClient, err := docker.NewClient(log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating docker client: %v", err)
	}

	// Get image tag
	imageTag, err := randutil.GenerateRandomString(7)
	if err != nil {
		t.Fatalf("Generating imageTag failed: %v", err)
	}

	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	// 2. Build image
	// 3. Don't push image
	imageName := "testimage"
	network := "someNetwork"
	buildArgs := make(map[string]*string)
	imageConfig := &latest.ImageConfig{
		Image: imageName,
		Build: &latest.BuildConfig{
			Docker: &latest.DockerConfig{
				Options: &latest.BuildOptions{
					BuildArgs: buildArgs,
					Network:   network,
				},
			},
		},
	}
	imageBuilder, err := NewBuilder(testConfig, dockerClient, kubeClient, imageName, imageConfig, imageTag, true, true)
	if err != nil {
		t.Fatalf("Builder creation failed: %v", err)
	}

	err = imageBuilder.BuildImage(dir, "Dockerfile", nil, nil, log.GetInstance())
	if err != nil {
		t.Fatalf("Image building failed: %v", err)
	}

}

func TestDockerbuildWithEntryppointOverride(t *testing.T) {
	t.Skip("For some reason there's a problem with the coverage in this package")

	// 1. Write test dockerfile and context to a temp folder
	dir, err := ioutil.TempDir("", "testDocker")
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

	// 4. Cleanup temp folder
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

	deployConfig := &latest.DeploymentConfig{
		Name: "test-deployment",
	}

	// Create fake devspace config
	testConfig := &latest.Config{
		Deployments: []*latest.DeploymentConfig{
			deployConfig,
		},
		// The images config will tell the deployment method to override the image name used in the component above with the tag defined in the generated config below
		Images: map[string]*latest.ImageConfig{
			"default": &latest.ImageConfig{
				Image: "nginx",
			},
		},
	}
	loader.SetFakeConfig(testConfig)

	dockerClient, err := docker.NewClient(log.GetInstance())
	if err != nil {
		t.Fatalf("Error creating docker client: %v", err)
	}

	// Get image tag
	imageTag, err := randutil.GenerateRandomString(7)
	if err != nil {
		t.Fatalf("Generating imageTag failed: %v", err)
	}

	kubeClient := &kubectl.Client{
		Client: fake.NewSimpleClientset(),
	}

	// 2. Build image with entrypoint override (see parameter entrypoint in BuildImage)
	// 3. Don't push image
	imageName := "testimage"
	imageConfig := &latest.ImageConfig{
		Image: imageName,
	}
	imageBuilder, err := NewBuilder(testConfig, dockerClient, kubeClient, imageName, imageConfig, imageTag, true, true)
	if err != nil {
		t.Fatalf("Builder creation failed: %v", err)
	}

	err = imageBuilder.BuildImage(dir, "Dockerfile", []string{"node"}, []string{"index.js"}, log.GetInstance())
	if err != nil {
		t.Fatalf("Image building failed: %v", err)
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
