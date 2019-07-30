package generator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/devspace-cloud/devspace/pkg/util/fsutil"
	"github.com/devspace-cloud/devspace/pkg/util/git"
	homedir "github.com/mitchellh/go-homedir"
)

// ChartRepoURL is the repository url
const ChartRepoURL = "https://github.com/devspace-cloud/helm-chart.git"

// ChartRepoPath is the path relative to the user folder where the chart repo is stored
const ChartRepoPath = ".devspace/chartRepo"

// ChartGenerator holds the information to create a chart and update a chart
type ChartGenerator struct {
	LocalPath string
	gitRepo   *git.Repository
}

// NewChartGenerator creates a new chart generator for the given path
func NewChartGenerator(localPath string) (*ChartGenerator, error) {
	if localPath == "" {
		localPath = "."
	}

	homedir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	gitRepository := git.NewGitRepository(filepath.Join(homedir, ChartRepoPath), ChartRepoURL)
	return &ChartGenerator{
		LocalPath: localPath,
		gitRepo:   gitRepository,
	}, nil
}

// Update updates the chart if already exists or creates a new chart if not
func (cg *ChartGenerator) Update(force bool) error {
	err := cg.gitRepo.Update(true)
	if err != nil {
		return fmt.Errorf("Error updating repository: %v", err)
	}

	// Check if chart folder already exists
	_, err = os.Stat(cg.LocalPath)
	if err == nil {
		// Check if everything is correct
		_, err := os.Stat(filepath.Join(cg.LocalPath, "devspace.yaml"))
		if force == false && os.IsNotExist(err) {
			return fmt.Errorf("Error updating chart: Chart at %s is not a devspace-chart, you can force the update with `devspace update chart --force`", cg.LocalPath)
		}
	} else {
		err = os.MkdirAll(cg.LocalPath, 0755)
		if err != nil {
			return err
		}
	}

	// Create templates folder if does not exist
	localTemplatesFolder := filepath.Join(cg.LocalPath, "templates")
	err = os.MkdirAll(localTemplatesFolder, 0755)
	if err != nil {
		return err
	}

	// Clean templates folder
	err = cleanTemplatesFolder(localTemplatesFolder)
	if err != nil {
		return err
	}

	// Copy into the chart folder
	err = fsutil.Copy(filepath.Join(cg.gitRepo.LocalPath, "chart"), cg.LocalPath, false)
	if err != nil {
		return err
	}

	return nil
}

func cleanTemplatesFolder(templatesFolder string) error {
	files, err := ioutil.ReadDir(templatesFolder)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() == false {
			err = os.Remove(filepath.Join(templatesFolder, f.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
