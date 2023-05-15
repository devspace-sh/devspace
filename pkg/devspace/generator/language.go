package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/survey"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	detector "github.com/loft-sh/programming-language-detection/pkg/detector"
)

// DevSpaceContainerRepo is the default repository url
const DevSpaceContainerRepo = "loft-sh/devspace-containers"
const GithubContainerRegistry = "ghcr.io"

// devContainersRepoPath is the path relative to the user folder where the docker file repo is stored
const devContainersRepoPath = ".devspace/devspace-containers"

const langCSharpDotNet = "c# (dotnet)"
const langFallback = "alpine"

// LanguageHandler is a type of object that generates a Helm Chart
type LanguageHandler struct {
	Language  string
	LocalPath string

	gitRepo            *git.GoGitRepository
	supportedLanguages []string

	log log.Logger
}

// NewLanguageHandler creates a new dockerfile generator
func NewLanguageHandler(localPath, templateRepoURL string, log log.Logger) (*LanguageHandler, error) {
	repoURL := fmt.Sprintf("https://github.com/%s.git", DevSpaceContainerRepo)
	if templateRepoURL != "" {
		repoURL = templateRepoURL
	}

	homedir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	gitRepository := git.NewGoGitRepository(filepath.Join(homedir, devContainersRepoPath), repoURL)

	return &LanguageHandler{
		LocalPath: localPath,
		gitRepo:   gitRepository,
		log:       log,
	}, nil
}

func (cg *LanguageHandler) GetDevImage() (string, error) {
	language, err := cg.GetLanguage()
	if err != nil {
		return "", err
	}

	reader, err := os.OpenFile(filepath.Join(cg.gitRepo.LocalPath, language, "Dockerfile"), os.O_RDONLY, 0600)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	tag := "latest"
	sc := bufio.NewScanner(reader)
	if sc.Scan() {
		argParts := strings.Split(sc.Text(), "=")
		if len(argParts) > 1 {
			tag = argParts[1]
		}
	}

	return fmt.Sprintf("%s/%s/%s:%s", GithubContainerRegistry, DevSpaceContainerRepo, language, tag), nil
}

func (cg *LanguageHandler) CopyTemplates(targetPath string, overwrite bool) error {
	language, err := cg.GetLanguage()
	if err != nil {
		return err
	}

	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	return fsutil.Copy(filepath.Join(cg.gitRepo.LocalPath, language, "template"), absTargetPath, overwrite)
}

func (cg *LanguageHandler) CopyFile(fileName, targetPath string, overwrite bool) error {
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	_, err = os.Stat(absTargetPath)
	if err != nil || overwrite {
		language, err := cg.GetLanguage()
		if err != nil {
			return err
		}

		err = fsutil.Copy(filepath.Join(cg.gitRepo.LocalPath, language, "template", fileName), absTargetPath, overwrite)
		if err != nil {
			return err
		}
	}

	// Ensure file is executable
	err = os.Chmod(absTargetPath, 0755)
	if err != nil {
		return err
	}

	return nil
}

// GetLanguage gets the language from LanguageHandler either from its field "Language" or by detecting it
func (cg *LanguageHandler) GetLanguage() (string, error) {
	// If the language was determined already, return it from cache
	if cg.Language != "" {
		return cg.Language, nil
	}

	cg.log.WriteString(logrus.WarnLevel, "\n")
	cg.log.Info("Detecting programming language...")

	language, detectionErr := cg.detectLanguage()
	if detectionErr != nil {
		return "", detectionErr
	}

	supportedLanguages, err := cg.GetSupportedLanguages()
	if err != nil {
		cg.log.Warnf("Error retrieving support languages: %v", err)
	}

	if len(supportedLanguages) == 0 {
		language = langFallback
	} else {
		otherOption := "other"
		if language == langFallback {
			language = otherOption
		}

		// Let the user select the language
		language, err = cg.log.Question(&survey.QuestionOptions{
			Question:     "Select the programming language of this project",
			DefaultValue: language,
			Options:      append(supportedLanguages, otherOption),
		})
		if err != nil {
			return "", err
		}

		if language == otherOption {
			language = langFallback
		}
	}

	language = regexp.MustCompile(`^.*\((.*)\)$`).ReplaceAllString(language, "$1")

	// Save user's choice in cache
	cg.Language = language

	return cg.Language, nil
}

// IsSupportedLanguage returns true if the given language is supported by the LanguageHandler
func (cg *LanguageHandler) IsSupportedLanguage(language string) (bool, string) {
	supportedLanguages, _ := cg.GetSupportedLanguages()

	if language == "dotnet" {
		return true, langCSharpDotNet
	}

	for _, supportedLanguage := range supportedLanguages {
		if language == supportedLanguage || strings.HasPrefix(supportedLanguage, language+"-") {
			return true, supportedLanguage
		}
	}
	return false, ""
}

// GetSupportedLanguages returns all languages that are available in the local Template Rempository
func (cg *LanguageHandler) GetSupportedLanguages() ([]string, error) {
	err := cg.gitRepo.Update(true)
	if err != nil {
		// try to remove and re-clone
		_ = os.RemoveAll(cg.gitRepo.LocalPath)
		err = cg.gitRepo.Update(true)
		if err != nil {
			return nil, errors.Errorf("Error updating git repo %s: %v", cg.gitRepo.RemoteURL, err)
		}
	}

	if len(cg.supportedLanguages) == 0 {
		files, err := os.ReadDir(cg.gitRepo.LocalPath)
		if err != nil {
			return nil, err
		}

		for _, dirEntry := range files {
			file, err := dirEntry.Info()
			if err != nil {
				continue
			}

			fileName := file.Name()

			if file.IsDir() && fileName[0] != '_' && fileName[0] != '.' && fileName != langFallback {
				if fileName == "dotnet" {
					fileName = langCSharpDotNet
				}
				cg.supportedLanguages = append(cg.supportedLanguages, fileName)
			}
		}
	}

	return cg.supportedLanguages, nil
}

// CreateDockerfile creates a dockerfile for a given language
func (cg *LanguageHandler) CreateDockerfile(language string) error {
	err := cg.gitRepo.Update(true)
	if err != nil {
		return err
	}

	// Check if language is available
	_, err = os.Stat(filepath.Join(cg.gitRepo.LocalPath, language))
	if err != nil {
		return errors.Errorf("Template for language %s not found", language)
	}

	// Copy dockerfile
	err = fsutil.Copy(filepath.Join(cg.gitRepo.LocalPath, language), ".", false)
	if err != nil {
		return err
	}

	return nil
}

func (cg *LanguageHandler) detectLanguage() (string, error) {
	contentReadLimit := 16 * 1024 * 1024

	detectedLanguage := detector.GetLanguage(".", contentReadLimit)
	detectedLanguage = strings.ToLower(detectedLanguage)

	isSupported, language := cg.IsSupportedLanguage(detectedLanguage)
	if !isSupported {
		language = langFallback
	}

	cg.Language = language

	return cg.Language, nil
}
