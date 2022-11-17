package generator

import (
	"os"
	"regexp"
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
	supported, _ := languageHandler.IsSupportedLanguage("unsupportedLanguage")
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

	match := regexp.MustCompile(`ARG  TAG=\d*-alpine
FROM node:\$\{TAG\}

# Set working directory
WORKDIR /app
`)
	assert.Assert(t, match.Match(content), "Created Dockerfile has wrong content")

	//Test CreateDockerFile with unavailable language
	err = languageHandler.CreateDockerfile("unavailableLanguage")
	if err == nil {
		t.Fatalf("No Error creating Dockerfile from languageHandler with unavailable Language: %v", err)
	}

}
