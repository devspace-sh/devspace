package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/loft-sh/devspace/pkg/util/encoding"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// DependencyFolder is the dependency folder in the home directory of the user
const DependencyFolder = ".devspace/dependencies"

// DependencyFolderPath will be filled during init
var DependencyFolderPath string

func init() {
	// Make sure dependency folder exists locally
	homedir, _ := homedir.Dir()

	DependencyFolderPath = filepath.Join(homedir, filepath.FromSlash(DependencyFolder))
}

// downloadMutex makes sure we only download a single dependency at a time
var downloadMutex = sync.Mutex{}

func GetDependencyPath(workingDirectory string, source *latest.SourceConfig) (configPath string, err error) {
	ID, err := GetDependencyID(source)
	if err != nil {
		return "", err
	}

	// Resolve source
	var localPath string
	if source.Git != "" {
		localPath = filepath.Join(DependencyFolderPath, ID)
	} else if source.Path != "" {
		if isURL(source.Path) {
			localPath = filepath.Join(DependencyFolderPath, ID)
		} else {
			if filepath.IsAbs(source.Path) {
				localPath = source.Path
			} else {
				localPath, err = filepath.Abs(filepath.Join(workingDirectory, filepath.FromSlash(source.Path)))
				if err != nil {
					return "", errors.Wrap(err, "filepath absolute")
				}
			}
		}
	}

	return getDependencyConfigPath(localPath, source)
}

// switch https <-> ssh  urls
func switchURLType(gitPath string) string {
	var newGitURL string
	if strings.HasPrefix(gitPath, "https") {
		splitURL := strings.Split(gitPath, "/")
		newGitURL = fmt.Sprintf("git@%s:%s", splitURL[2], strings.Join(splitURL[3:], "/"))
	} else {
		splitURL := strings.Split(gitPath, "@")
		replacedURL := strings.ReplaceAll(splitURL[1], ":", "/")
		newGitURL = fmt.Sprintf("https://%s", replacedURL)
	}
	return newGitURL
}

func DownloadDependency(ctx context.Context, workingDirectory string, source *latest.SourceConfig, log log.Logger) (configPath string, err error) {
	downloadMutex.Lock()
	defer downloadMutex.Unlock()

	ID, err := GetDependencyID(source)
	if err != nil {
		return "", err
	}

	var localPath string

	// Resolve git source
	if source.Git != "" {
		gitPath := strings.TrimSpace(source.Git)

		_ = os.MkdirAll(DependencyFolderPath, 0755)
		localPath = filepath.Join(DependencyFolderPath, ID)

		// Check if dependency are cached locally
		_, statErr := os.Stat(localPath)

		// Verify git cli works
		repo, err := git.NewGitCLIRepository(ctx, localPath)
		if err != nil {
			if statErr == nil {
				log.Warnf("Error creating git cli: %v", err)
				return getDependencyConfigPath(localPath, source)
			}
			return "", err
		}

		// Create git clone options
		var gitCloneOptions = git.CloneOptions{
			URL:            gitPath,
			Tag:            source.Tag,
			Branch:         source.Branch,
			Commit:         source.Revision,
			Args:           source.CloneArgs,
			DisableShallow: source.DisableShallow,
		}

		// Git clone
		if statErr != nil {
			err = repo.Clone(ctx, gitCloneOptions)

			if err != nil {
				log.Warn("Error cloning repo: ", err)

				gitCloneOptions.URL = switchURLType(gitPath)
				log.Infof("Switching URL from %s to %s and will try cloning again", gitPath, gitCloneOptions.URL)
				err = repo.Clone(ctx, gitCloneOptions)

				if err != nil {
					log.Warn("Failed to clone repo with both HTTPS and SSH URL. Please make sure if your git login or ssh setup is correct.")
					if statErr == nil {
						log.Warnf("Error cloning or pulling git repository %s: %v", gitPath, err)
						return getDependencyConfigPath(localPath, source)
					}

					return "", errors.Wrap(err, "clone repository")
				}
			}

			log.Debugf("Cloned %s", gitPath)
		}

		// Git pull
		if !source.DisablePull && source.Revision == "" {
			err = repo.Pull(ctx)
			if err != nil {
				log.Warn(err)
			}

			log.Debugf("Pulled %s", gitPath)
		}

		// Resolve local source
	} else if source.Path != "" {
		if isURL(source.Path) {
			localPath = filepath.Join(DependencyFolderPath, ID)
			_ = os.MkdirAll(localPath, 0755)

			// Check if dependency exists
			configPath := filepath.Join(localPath, constants.DefaultConfigPath)
			_, statErr := os.Stat(configPath)

			if !source.DisablePull || statErr != nil {
				// Create the file
				out, err := os.Create(configPath)
				if err != nil {
					if statErr == nil {
						log.Warnf("Error creating file: %v", err)
						return getDependencyConfigPath(localPath, source)
					}

					return "", err
				}
				defer out.Close()

				// Get the data
				resp, err := http.Get(source.Path)
				if err != nil {
					if statErr == nil {
						log.Warnf("Error retrieving url %s: %v", source.Path, err)
						return getDependencyConfigPath(localPath, source)
					}

					return "", errors.Wrapf(err, "request %s", source.Path)
				}
				defer resp.Body.Close()

				// Write the body to file
				_, err = io.Copy(out, resp.Body)
				if err != nil {
					if statErr == nil {
						log.Warnf("Error retrieving url %s: %v", source.Path, err)
						return getDependencyConfigPath(localPath, source)
					}

					return "", errors.Wrapf(err, "download %s", source.Path)
				}
			}
		} else {
			if filepath.IsAbs(source.Path) {
				localPath = source.Path
			} else {
				localPath, err = filepath.Abs(filepath.Join(workingDirectory, filepath.FromSlash(source.Path)))
				if err != nil {
					return "", errors.Wrap(err, "filepath absolute")
				}
			}
		}
	}

	return getDependencyConfigPath(localPath, source)
}

func getDependencyConfigPath(dependencyPath string, source *latest.SourceConfig) (string, error) {
	var configPath string
	if source.SubPath != "" {
		dependencyPath = filepath.Join(dependencyPath, filepath.FromSlash(source.SubPath))
	}
	if strings.HasSuffix(dependencyPath, ".yaml") || strings.HasSuffix(dependencyPath, ".yml") {
		configPath = dependencyPath
	} else {
		configPath = filepath.Join(dependencyPath, constants.DefaultConfigPath)
	}

	return configPath, nil
}

func GetDependencyID(source *latest.SourceConfig) (string, error) {
	// check if source is there
	if source == nil {
		return "", fmt.Errorf("source is missing")
	}

	// get id for git
	if source.Git != "" {
		id := source.Git
		if source.Branch != "" {
			id += "@" + source.Branch
		} else if source.Tag != "" {
			id += "@tag:" + source.Tag
		} else if source.Revision != "" {
			id += "@revision:" + source.Revision
		}

		return encoding.Convert(id), nil
	} else if source.Path != "" {
		return source.Path, nil
	}

	return "", fmt.Errorf("unexpected dependency config, both source.git and source.path are missing")
}

func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
