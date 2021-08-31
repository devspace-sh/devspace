package util

import (
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/hash"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var authRegEx = regexp.MustCompile("^(https?:\\/\\/)[^:]+:[^@]+@(.*)$")

// DependencyFolder is the dependency folder in the home directory of the user
const DependencyFolder = ".devspace/dependencies"

// DependencyFolderPath will be filled during init
var DependencyFolderPath string

func init() {
	// Make sure dependency folder exists locally
	homedir, _ := homedir.Dir()

	DependencyFolderPath = filepath.Join(homedir, filepath.FromSlash(DependencyFolder))
}

func DownloadDependency(ID, basePath string, source *latest.SourceConfig, update bool, log log.Logger) (localPath string, err error) {
	// Resolve source
	if source.Git != "" {
		gitPath := strings.TrimSpace(source.Git)

		os.MkdirAll(DependencyFolderPath, 0755)
		localPath = filepath.Join(DependencyFolderPath, hash.String(ID))

		// Check if dependency exists
		_, err := os.Stat(localPath)
		if err != nil {
			update = true
		}

		// Update dependency
		if update {
			repo, err := git.NewGitCLIRepository(localPath)
			if err != nil {
				return "", err
			}

			err = repo.Clone(git.CloneOptions{
				URL:            gitPath,
				Tag:            source.Tag,
				Branch:         source.Branch,
				Commit:         source.Revision,
				Args:           source.CloneArgs,
				DisableShallow: source.DisableShallow,
			})
			if err != nil {
				return "", errors.Wrap(err, "clone repository")
			}

			log.Donef("Pulled %s", ID)
		}
	} else if source.Path != "" {
		if isUrl(source.Path) {
			localPath = filepath.Join(DependencyFolderPath, hash.String(ID))
			os.MkdirAll(localPath, 0755)

			// Check if dependency exists
			configPath := filepath.Join(localPath, constants.DefaultConfigPath)
			_, err := os.Stat(configPath)
			if err != nil {
				update = true
			}

			if update {
				// Create the file
				out, err := os.Create(configPath)
				if err != nil {
					return "", err
				}
				defer out.Close()

				// Get the data
				resp, err := http.Get(source.Path)
				if err != nil {
					return "", errors.Wrapf(err, "request %s", source.Path)
				}
				defer resp.Body.Close()

				// Write the body to file
				_, err = io.Copy(out, resp.Body)
				if err != nil {
					return "", errors.Wrapf(err, "download %s", source.Path)
				}
			}
		} else {
			if filepath.IsAbs(source.Path) {
				localPath = source.Path
			} else {
				localPath, err = filepath.Abs(filepath.Join(basePath, filepath.FromSlash(source.Path)))
				if err != nil {
					return "", errors.Wrap(err, "filepath absolute")
				}
			}
		}
	}

	if source.SubPath != "" {
		localPath = filepath.Join(localPath, filepath.FromSlash(source.SubPath))
	}

	return localPath, nil
}

func GetDependencyID(basePath string, config *latest.DependencyConfig) (string, error) {
	out, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return hash.String(basePath + ";" + string(out)), nil
}

func GetParentProfileID(basePath string, source *latest.SourceConfig, profile string, vars []latest.DependencyVar) string {
	if source.Git != "" {
		// Erase authentication credentials
		id := strings.TrimSpace(source.Git)
		id = authRegEx.ReplaceAllString(id, "$1$2")

		if source.Tag != "" {
			id += "@" + source.Tag
		} else if source.Branch != "" {
			id += "@" + source.Branch
		} else if source.Revision != "" {
			id += "@" + source.Revision
		}
		if source.SubPath != "" {
			id += ":" + source.SubPath
		}
		if profile != "" {
			id += " - profile " + profile
		}
		if len(source.CloneArgs) > 0 {
			id += " - with clone args " + strings.Join(source.CloneArgs, " ")
		}
		for _, v := range vars {
			id += ";" + v.Name + "=" + v.Value
		}

		return id
	} else if source.Path != "" {
		if isUrl(source.Path) {
			id := strings.TrimSpace(source.Path)

			if profile != "" {
				id += " - profile " + profile
			}
			for _, v := range vars {
				id += ";" + v.Name + "=" + v.Value
			}

			return id
		}

		// Check if it's an git repo
		filePath := source.Path
		if !filepath.IsAbs(source.Path) {
			filePath = filepath.Join(basePath, source.Path)
		}

		remote, err := git.GetRemote(filePath)
		if err == nil {
			return remote
		}

		if source.ConfigName != "" {
			filePath += filepath.Join(filePath, source.ConfigName)
		}

		if profile != "" {
			filePath += " - profile " + profile
		}

		for _, v := range vars {
			filePath += ";" + v.Name + "=" + v.Value
		}

		return filePath
	}

	return ""
}

func isUrl(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
