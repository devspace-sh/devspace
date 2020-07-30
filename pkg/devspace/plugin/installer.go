package plugin

import (
	"github.com/devspace-cloud/devspace/pkg/util/git"
	"github.com/ghodss/yaml"
	"github.com/otiai10/copy"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Installer interface {
	DownloadMetadata(path, version string) (*Metadata, error)
	DownloadBinary(metadataPath, version, binaryPath, outFile string) error
}

type installer struct {}

func NewInstaller() Installer {
	return &installer{}
}

func (i *installer) DownloadBinary(metadataPath, version, binaryPath, outFile string) error {
	if isLocalReference(metadataPath) {
		localPath := filepath.Join(filepath.Dir(metadataPath), binaryPath)
		if isLocalReference(localPath) {
			return copy.Copy(localPath, outFile)
		}

		return i.downloadTo(binaryPath, outFile)
	} else if isRemoteHTTPYAML(metadataPath) {
		return i.downloadTo(binaryPath, outFile)
	}

	if strings.HasPrefix(binaryPath, "http://") || strings.HasPrefix(binaryPath, "https://") {
		return i.downloadTo(binaryPath, outFile)
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempDir)
	repo, err := git.NewGitCLIRepository(tempDir)
	if err != nil {
		return err
	}

	err = repo.Clone(git.CloneOptions{
		URL: metadataPath,
		Tag: version,
	})
	if err != nil {
		return err
	}

	return copy.Copy(filepath.Join(tempDir, binaryPath), outFile)
}

func (i *installer) downloadTo(binaryPath, outFile string) error {
	resp, err := http.Get(binaryPath)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func (i *installer) DownloadMetadata(path, version string) (*Metadata, error) {
	if isLocalReference(path) {
		out, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		metadata := &Metadata{}
		err = yaml.Unmarshal(out, metadata)
		if err != nil {
			return nil, err
		}

		return metadata, nil
	} else if isRemoteHTTPYAML(path) {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}

		out, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		metadata := &Metadata{}
		err = yaml.Unmarshal(out, metadata)
		if err != nil {
			return nil, err
		}

		return metadata, nil
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(tempDir)
	repo, err := git.NewGitCLIRepository(tempDir)
	if err != nil {
		return nil, err
	}

	err = repo.Clone(git.CloneOptions{
		URL:            path,
		Tag:            version,
	})
	if err != nil {
		return nil, err
	}

	out, err := ioutil.ReadFile(filepath.Join(tempDir, pluginYaml))
	if err != nil {
		return nil, err
	}

	metadata := &Metadata{}
	err = yaml.Unmarshal(out, metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// isLocalReference checks if the source exists on the filesystem.
func isLocalReference(source string) bool {
	_, err := os.Stat(source)
	return err == nil
}

// isRemoteHTTPYAML checks if the source is a http/https url and a yaml
func isRemoteHTTPYAML(source string) bool {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		if strings.HasSuffix(source, ".yaml") {
			return true
		}
	}
	return false
}