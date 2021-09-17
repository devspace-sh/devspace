package generator

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/fsutil"
	fakelogger "github.com/loft-sh/devspace/pkg/util/log/testing"
	"gotest.tools/assert"
)

func TestContainerizeApplication(t *testing.T) {
	//Create TmpFolder
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

	// Cleanup temp folder
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

	err = fsutil.WriteToFile([]byte(`var express = require('express');
var app = express();

app.get('/', function (req, res) {
  res.send('Hello World!');
});

app.listen(3000, function () {
  console.log('Example app listening on port 3000!');
});`), "index.js")
	if err != nil {
		t.Fatalf("Error creating javascript file: %v", err)
	}

	fakeLogger := fakelogger.NewFakeLogger()
	fakeLogger.Survey.SetNextAnswer("javascript")

	generator, err := NewDockerfileGenerator("", "", fakeLogger)
	if err != nil {
		t.Fatalf("Error containerizing application: %v", err)
	}

	err = generator.ContainerizeApplication("")
	if err != nil {
		t.Fatalf("Error containerizing application: %v", err)
	}
}

func TestDockerfileGenerator(t *testing.T) {
	//Create TmpFolder
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

	// Cleanup temp folder
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

	//Test factory method
	dockerfileGenerator, err := NewDockerfileGenerator("", "", fakelogger.NewFakeLogger())
	if err != nil {
		t.Fatalf("Error creating a dockerfileGenerator: %v", err)
	}

	t.Log(dockerfileGenerator.gitRepo.LocalPath)

	//Test IsLanguageSupported with unsupported Language
	supported := dockerfileGenerator.IsSupportedLanguage("unsupportedLanguage")
	assert.Equal(t, false, supported, "Unsupported language is declared supported.")

	//Test CreateDockerFile
	err = dockerfileGenerator.CreateDockerfile("javascript")
	if err != nil {
		t.Fatalf("Error creating Dockerfile from dockerfileGenerator: %v", err)
	}
	content, err := fsutil.ReadFile("Dockerfile", -1)
	if err != nil {
		t.Fatalf("Error reading Dockerfile. Maybe dockerfileGenerator.CreateDockerfile didn't create it? : %v", err)
	}
	fmt.Println(string(content))
	assert.Equal(t, string(content), `FROM node:13.14-alpine

# Set working directory
WORKDIR /app

# Add package.json to WORKDIR and install dependencies
COPY package*.json ./
RUN npm install

# Add source code files to WORKDIR
COPY . .

# Application port (optional)
EXPOSE 3000

# Debugging port (optional)
# For remote debugging, add this port to devspace.yaml: dev.ports[*].forward[*].port: 9229
EXPOSE 9229

# Container start command (DO NOT CHANGE and see note below)
CMD ["npm", "start"]

# To start using a different `+"`npm run [name]` "+`command (e.g. to use nodemon + debugger),
# edit devspace.yaml:
# 1) remove: images.app.injectRestartHelper (or set to false)
# 2) add this: images.app.cmd: ["npm", "run", "dev"]
`, "Created Dockerfile has wrong content")

	//Test CreateDockerFile with unavailable language
	err = dockerfileGenerator.CreateDockerfile("unavailableLanguage")
	if err == nil {
		t.Fatalf("No Error creating Dockerfile from dockerfileGenerator with unavailable Language: %v", err)
	}

}
