package generator

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/covexo/devspace/pkg/util/fsutil"

	"gopkg.in/src-d/enry.v1"
	"gopkg.in/src-d/go-git.v4"
)

// ChartGenerator is a type of object that generates a Helm Chart
type ChartGenerator struct {
	Path               string
	Language           string
	TemplateRepo       *TemplateRepository
	supportedLanguages []string
}

type TemplateRepository struct {
	URL       string
	LocalPath string
}

// GetLanguage gets the language from Chartgenerator either from its field "Language" or by detecting it
func (cg *ChartGenerator) GetLanguage() (string, error) {
	if len(cg.Language) == 0 {
		detectionErr := cg.detectLanguage()

		if detectionErr != nil {
			return "", detectionErr
		}
	}
	return cg.Language, nil
}

// IsSupportedLanguage returns true if the given language is supported by the ChartGenerator
func (cg *ChartGenerator) IsSupportedLanguage(language string) bool {
	supportedLanguages, _ := cg.GetSupportedLanguages()

	for _, supportedLanguage := range supportedLanguages {
		if language == supportedLanguage {
			return true
		}
	}
	return false
}

// GetSupportedLanguages returns all languages that are available in the local Template Rempository
func (cg *ChartGenerator) GetSupportedLanguages() ([]string, error) {
	chartCloneErr := cg.getChartTemplates()

	if chartCloneErr != nil {
		return nil, chartCloneErr
	}

	if len(cg.supportedLanguages) == 0 {
		files, err := ioutil.ReadDir(cg.TemplateRepo.LocalPath)

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

func (cg *ChartGenerator) CreateChart() error {
	chartUpdateError := cg.getChartTemplates()

	if chartUpdateError != nil {
		return chartUpdateError
	}
	language, langError := cg.GetLanguage()

	if langError != nil {
		return langError
	}
	_, languageTemplateNotFound := os.Stat(cg.TemplateRepo.LocalPath + "/" + language)

	if languageTemplateNotFound != nil {
		return errors.New("Language Template not found")
	}
	copyBaseError := fsutil.Copy(cg.TemplateRepo.LocalPath+"/_base", cg.Path)

	if copyBaseError != nil {
		return copyBaseError
	}
	copyError := fsutil.Copy(cg.TemplateRepo.LocalPath+"/"+cg.Language, cg.Path)

	if copyError != nil {
		return copyError
	}
	return nil
}

func (cg *ChartGenerator) AddPackage(pkg string) {

}

func (cg *ChartGenerator) RemovePackage(pkg string) {

}

func (cg *ChartGenerator) getChartTemplates() error {
	_, repoNotFound := os.Stat(cg.TemplateRepo.LocalPath + "/.git")

	if repoNotFound == nil {
		repo, _ := git.PlainOpen(cg.TemplateRepo.LocalPath)
		repoWorktree, _ := repo.Worktree()

		repoWorktree.Pull(&git.PullOptions{
			RemoteName: "origin",
		})
		return nil
	} else {
		_, cloneErr := git.PlainClone(cg.TemplateRepo.LocalPath, false, &git.CloneOptions{
			URL: cg.TemplateRepo.URL,
		})
		return cloneErr
	}
}

func (cg *ChartGenerator) detectLanguage() error {
	contentReadLimit := int64(16 * 1024 * 1024)
	bytesByLanguage := make(map[string]int64, 0)

	// Cancel the language detection after 10sec
	cancelDetect := false
	time.AfterFunc(10*time.Second, func() {
		cancelDetect = true
	})

	walkError := filepath.Walk(cg.Path, func(path string, fileInfo os.FileInfo, err error) error {

		// If timeout is over, then cancel detect
		if cancelDetect {
			return filepath.SkipDir
		}

		if err != nil {
			log.Println(err)
			return filepath.SkipDir
		}

		if !fileInfo.Mode().IsDir() && !fileInfo.Mode().IsRegular() {
			return nil
		}

		relativePath, err := filepath.Rel(cg.Path, path)
		if err != nil {
			log.Println(err)
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
					log.Println(err)
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
		if bytes > currentMaxBytes {
			detectedLanguage = language
			currentMaxBytes = bytes
		}
	}
	detectedLanguage = strings.ToLower(detectedLanguage)

	if cg.IsSupportedLanguage(detectedLanguage) {
		cg.Language = detectedLanguage
	}
	return nil
}
