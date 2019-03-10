package generator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/stdinutil"
	homedir "github.com/mitchellh/go-homedir"

	enry "gopkg.in/src-d/enry.v1"
)

// DockerfileRepoURL is the default repository url
const DockerfileRepoURL = "https://github.com/devspace-cloud/devspace-templates.git"

// DockerfileRepoPath is the path relative to the user folder where the docker file repo is stored
const DockerfileRepoPath = ".devspace/dockerfileRepo"

// DockerfileGenerator is a type of object that generates a Helm Chart
type DockerfileGenerator struct {
	Language  string
	LocalPath string

	gitRepo            *GitRepository
	supportedLanguages []string
}

// ContainerizeApplication will create a dockerfile at the given path based on the language detected
func ContainerizeApplication(localPath string, templateRepoURL string) error {
	// Check if the user already has a dockerfile
	_, err := os.Stat(filepath.Join("Dockerfile"))
	if os.IsNotExist(err) == false {
		log.Infof("Devspace will use the dockerfile at ./Dockerfile for building an image")
		return nil
	}

	var repoURL *string
	if templateRepoURL != "" {
		repoURL = &templateRepoURL
	}

	// Create new dockerfile generator
	dockerfileGenerator, err := NewDockerfileGenerator(localPath, repoURL)
	if err != nil {
		return err
	}

	log.StartWait("Detecting programming language")

	detectedLang := "none"
	supportedLanguages, err := dockerfileGenerator.GetSupportedLanguages()
	if err == nil {
		detectedLang, _ = dockerfileGenerator.GetLanguage()
		if detectedLang == "" {
			detectedLang = "none"
		}
	}
	if len(supportedLanguages) == 0 {
		supportedLanguages = []string{"none"}
	}

	log.StopWait()

	// Let the user select the language
	selectedLanguage := *stdinutil.GetFromStdin(&stdinutil.GetFromStdinParams{
		Question:     "Select programming language of project",
		DefaultValue: detectedLang,
		Options:      supportedLanguages,
	})

	return dockerfileGenerator.CreateDockerfile(selectedLanguage)
}

// NewDockerfileGenerator creates a new dockerfile generator
func NewDockerfileGenerator(localPath string, templateRepoURL *string) (*DockerfileGenerator, error) {
	repoURL := DockerfileRepoPath
	if templateRepoURL != nil {
		repoURL = *templateRepoURL
	}

	homedir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	gitRepository := NewGitRepository(filepath.Join(homedir, DockerfileRepoPath), repoURL)

	return &DockerfileGenerator{
		LocalPath: localPath,
		gitRepo:   gitRepository,
	}, nil
}

// GetLanguage gets the language from DockerfileGenerator either from its field "Language" or by detecting it
func (cg *DockerfileGenerator) GetLanguage() (string, error) {
	if len(cg.Language) == 0 {
		detectionErr := cg.detectLanguage()
		if detectionErr != nil {
			return "", detectionErr
		}
	}

	return cg.Language, nil
}

// IsSupportedLanguage returns true if the given language is supported by the DockerfileGenerator
func (cg *DockerfileGenerator) IsSupportedLanguage(language string) bool {
	supportedLanguages, _ := cg.GetSupportedLanguages()

	for _, supportedLanguage := range supportedLanguages {
		if language == supportedLanguage {
			return true
		}
	}
	return false
}

// GetSupportedLanguages returns all languages that are available in the local Template Rempository
func (cg *DockerfileGenerator) GetSupportedLanguages() ([]string, error) {
	_, err := cg.gitRepo.Update()
	if err != nil {
		return nil, fmt.Errorf("Error updating git repo %s: %v", cg.gitRepo.RemotURL, err)
	}

	if len(cg.supportedLanguages) == 0 {
		files, err := ioutil.ReadDir(cg.gitRepo.LocalPath)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			fileName := file.Name()

			if file.IsDir() && fileName[0] != '_' && fileName[0] != '.' {
				cg.supportedLanguages = append(cg.supportedLanguages, fileName)
			}
		}
	}
	return cg.supportedLanguages, nil
}

// CreateDockerfile creates a dockerfile for a given language
func (cg *DockerfileGenerator) CreateDockerfile(language string) error {
	_, err := cg.gitRepo.Update()
	if err != nil {
		return err
	}

	// Check if language is available
	_, err = os.Stat(filepath.Join(cg.gitRepo.LocalPath, language))
	if err != nil {
		return fmt.Errorf("Template for language %s not found", language)
	}

	// Copy dockerfile
	err = fsutil.Copy(filepath.Join(cg.gitRepo.LocalPath, cg.Language), ".", false)
	if err != nil {
		return err
	}

	return nil
}

func (cg *DockerfileGenerator) detectLanguage() error {
	contentReadLimit := int64(16 * 1024 * 1024)
	bytesByLanguage := make(map[string]int64, 0)

	// Cancel the language detection after 10sec
	cancelDetect := false
	time.AfterFunc(10*time.Second, func() {
		cancelDetect = true
	})

	walkError := filepath.Walk(".", func(path string, fileInfo os.FileInfo, err error) error {
		// If timeout is over, then cancel detect
		if cancelDetect {
			return filepath.SkipDir
		}

		if err != nil {
			return filepath.SkipDir
		}

		if !fileInfo.Mode().IsDir() && !fileInfo.Mode().IsRegular() {
			return nil
		}

		relativePath, err := filepath.Rel(".", path)
		if err != nil {
			return nil
		}

		if relativePath == "." {
			return nil
		}

		if fileInfo.IsDir() {
			relativePath = relativePath + "/"
		}

		if enry.IsVendor(relativePath) || enry.IsDotFile(relativePath) || enry.IsDocumentation(relativePath) || enry.IsConfiguration(relativePath) {
			if fileInfo.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if fileInfo.IsDir() {
			return nil
		}

		language, ok := enry.GetLanguageByExtension(path)
		if !ok {
			if language, ok = enry.GetLanguageByFilename(path); !ok {
				content, err := fsutil.ReadFile(path, contentReadLimit)
				if err != nil {
					return nil
				}

				language = enry.GetLanguage(filepath.Base(path), content)
				if language == enry.OtherLanguage {
					return nil
				}
			}
		}
		_, langExists := bytesByLanguage[language]
		if !langExists {
			bytesByLanguage[language] = 0
		}

		bytesByLanguage[language] = bytesByLanguage[language] + fileInfo.Size()
		return nil
	})

	if walkError != nil {
		return walkError
	}

	detectedLanguage := ""
	currentMaxBytes := int64(0)
	for language, bytes := range bytesByLanguage {
		language = strings.ToLower(language)

		if cg.IsSupportedLanguage(language) && bytes > currentMaxBytes {
			detectedLanguage = language
			currentMaxBytes = bytes
		}
	}

	if cg.IsSupportedLanguage(detectedLanguage) {
		cg.Language = detectedLanguage
	}

	return nil
}
