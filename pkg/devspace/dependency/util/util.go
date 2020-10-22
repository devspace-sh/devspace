package util

import (
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/devspace-cloud/devspace/pkg/util/hash"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

func DownloadDependency(basePath string, source *latest.SourceConfig, profile string, update bool, log log.Logger) (ID string, localPath string, err error) {
	ID = GetDependencyID(basePath, source, profile)

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
				return "", "", err
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
				return "", "", errors.Wrap(err, "clone repository")
			}

			log.Donef("Pulled %s", ID)
		}
	} else if source.Path != "" {
		if filepath.IsAbs(source.Path) {
			localPath = source.Path
		} else {
			localPath, err = filepath.Abs(filepath.Join(basePath, filepath.FromSlash(source.Path)))
			if err != nil {
				return "", "", errors.Wrap(err, "filepath absolute")
			}
		}
	}

	if source.SubPath != "" {
		localPath = filepath.Join(localPath, filepath.FromSlash(source.SubPath))
	}

	return ID, localPath, nil
}

func GetDependencyID(basePath string, source *latest.SourceConfig, profile string) string {
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

		return id
	} else if source.Path != "" {
		// Check if it's an git repo
		filePath := source.Path
		if !filepath.IsAbs(source.Path) {
			filePath = filepath.Join(basePath, source.Path)
		}

		remote, err := git.GetRemote(filePath)
		if err == nil {
			return remote
		}

		if profile != "" {
			filePath += " - profile " + profile
		}

		return filePath
	}

	return ""
}
