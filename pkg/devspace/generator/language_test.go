package generator

import (
	"fmt"
	"os"
	"testing"

	"github.com/loft-sh/devspace/pkg/util/fsutil"
	fakelogger "github.com/loft-sh/devspace/pkg/util/log/testing"
	"gotest.tools/assert"
)

func TestLanguageHandler(t *testing.T) {
	//Create TmpFolder
	dir := t.TempDir()

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
	}()

	//Test factory method
	languageHandler, err := NewLanguageHandler("", "", fakelogger.NewFakeLogger())
	if err != nil {
		t.Fatalf("Error creating a languageHandler: %v", err)
	}

	t.Log(languageHandler.gitRepo.LocalPath)

	//Test IsLanguageSupported with unsupported Language
	supported := languageHandler.IsSupportedLanguage("unsupportedLanguage")
	assert.Equal(t, false, supported, "Unsupported language is declared supported.")

	//Test CreateDockerFile
	err = languageHandler.CreateDockerfile("javascript")
	if err != nil {
		t.Fatalf("Error creating Dockerfile from languageHandler: %v", err)
	}
	content, err := fsutil.ReadFile("Dockerfile", -1)
	if err != nil {
		t.Fatalf("Error reading Dockerfile. Maybe languageHandler.CreateDockerfile didn't create it? : %v", err)
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
	err = languageHandler.CreateDockerfile("unavailableLanguage")
	if err == nil {
		t.Fatalf("No Error creating Dockerfile from languageHandler with unavailable Language: %v", err)
	}

}
