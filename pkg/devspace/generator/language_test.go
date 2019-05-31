package generator

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	
	"gotest.tools/assert"
)

func TestContainerizeApplication(t *testing.T){
	t.Skip("Question-call interrupts test session, therefore skipped")

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
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

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


	//Fake stdin
    content := []byte("\r\n")
    tmpfile, err := ioutil.TempFile("", "stdin")
    if err != nil {
        t.Fatalf("Error creating temporary stdin file: %v", err)
    }

	defer tmpfile.Close()
    defer os.Remove(tmpfile.Name()) // clean up

    if _, err := tmpfile.Write(content); err != nil {
        t.Fatalf("Error writing temporary stdin file: %v", err)
    }

    if _, err := tmpfile.Seek(0, 0); err != nil {
        t.Fatalf("Error setting pointer in temporary stdin file: %v", err)
    }

    oldStdin := os.Stdin
    defer func() { os.Stdin = oldStdin }() // Restore original Stdin
    os.Stdin = tmpfile

	//err = ContainerizeApplication("", "", "")
	t.Log("App containerized")
	if err != nil {
		t.Fatalf("Error containerizing application: %v", err)
	}
	time.Sleep(time.Second * 10)
	t.Log("Finished")
}

func TestDockerfileGenerator(t *testing.T){
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
	defer os.Chdir(wdBackup)
	defer os.RemoveAll(dir)

	fsutil.WriteToFile([]byte(`var express = require('express');
var app = express();

app.get('/', function (req, res) {
  res.send('Hello World!');
});

app.listen(3000, function () {
  console.log('Example app listening on port 3000!');
});`), "someDir/index.js")
	if err != nil {
		t.Fatalf("Error creating javascript file: %v", err)
	}
	err = fsutil.WriteToFile([]byte(""), "deps/.dotFile")
	if err != nil {
		t.Fatalf("Error creating dotfile: %v", err)
	}
	err = fsutil.WriteToFile([]byte(""), ".dotFile")
	if err != nil {
		t.Fatalf("Error creating dotfile: %v", err)
	}

	//Test factory method
	dockerfileGenerator, err := NewDockerfileGenerator("", ptr.String(""))
	if err != nil {
		t.Fatalf("Error creating a dockerfileGenerator: %v", err)
	}
	
	//Test GetLanguage
	detectedLanguage, err := dockerfileGenerator.GetLanguage()
	if err != nil {
		t.Fatalf("Error getting language from dockerfileGenerator: %v", err)
	}
	assert.Equal(t, "javascript", detectedLanguage, "Wrong language detected")

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
