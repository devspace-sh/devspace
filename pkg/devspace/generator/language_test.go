package generator

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	fakelogger "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	"gotest.tools/assert"
)

func TestContainerizeApplicationWithExistingDockerfile(t *testing.T) {
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

	err = fsutil.WriteToFile([]byte(""), "Dockerfile")
	if err != nil {
		t.Fatalf("Error writing file: %v", err)
	}
	err = ContainerizeApplication("Dockerfile", "", "", log.GetInstance())
	assert.Error(t, err, "Dockerfile at Dockerfile already exists", "Unexpected or no error when trying to containerize with existing Dockerfile")
}

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
	err = ContainerizeApplication("", "", "", fakeLogger)
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
	dockerfileGenerator, err := NewDockerfileGenerator("", ptr.String(""))
	if err != nil {
		t.Fatalf("Error creating a dockerfileGenerator: %v", err)
	}

	t.Log(dockerfileGenerator.gitRepo.LocalPath)
	//dockerfileGenerator.gitRepo.LocalPath = "./gitLocal"
	err = fsutil.WriteToFile([]byte(`FROM node:8.11.4

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . .

CMD ["npm", "start"]
`), dockerfileGenerator.gitRepo.LocalPath+"/javascript/Dockerfile")
	if err != nil {
		t.Fatalf("Error writing to file: %v", err)
	}
	//err = fsutil.WriteToFile([]byte(`ref: refs/heads/master`), "gitLocal/javascript/.git/HEAD")
	//if err != nil {
	//t.Fatalf("Error writing to file: %v", err)
	//}

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
	assert.Equal(t, string(content), `FROM node:8.11.4

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . .

CMD ["npm", "start"]
`, "Created Dockerfile has wrong content")

	//Test CreateDockerFile with unavailable language
	err = dockerfileGenerator.CreateDockerfile("unavailableLanguage")
	if err == nil {
		t.Fatalf("No Error creating Dockerfile from dockerfileGenerator with unavailable Language: %v", err)
	}

}
