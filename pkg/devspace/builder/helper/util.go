package helper

import (
	"archive/tar"
	"errors"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/docker/docker/pkg/archive"
)

// DefaultDockerfilePath is the default dockerfile path to use
const DefaultDockerfilePath = "./Dockerfile"

// DefaultContextPath is the default context path to use
const DefaultContextPath = "./"

// GetDockerfileAndContext retrieves the dockerfile and context
func GetDockerfileAndContext(config *latest.Config, imageConfigName string, imageConf *latest.ImageConfig, isDev bool) (string, string) {
	var (
		dockerfilePath = DefaultDockerfilePath
		contextPath    = DefaultContextPath
	)

	if imageConf.Dockerfile != "" {
		dockerfilePath = imageConf.Dockerfile
	}

	if imageConf.Context != "" {
		contextPath = imageConf.Context
	}

	if isDev && config.Dev != nil && config.Dev.OverrideImages != nil {
		for _, overrideConfig := range config.Dev.OverrideImages {
			if overrideConfig.Name == imageConfigName {
				if overrideConfig.Dockerfile != "" {
					dockerfilePath = overrideConfig.Dockerfile
				}
				if overrideConfig.Context != "" {
					contextPath = overrideConfig.Context
				}
			}
		}
	}

	return dockerfilePath, contextPath
}

// OverwriteDockerfileInBuildContext will overwrite the dockerfile with the dockerfileCtx
func OverwriteDockerfileInBuildContext(dockerfileCtx io.ReadCloser, buildCtx io.ReadCloser, relDockerfile string) (io.ReadCloser, error) {
	file, err := ioutil.ReadAll(dockerfileCtx)
	dockerfileCtx.Close()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	hdrTmpl := &tar.Header{
		Mode:       0600,
		Uid:        0,
		Gid:        0,
		ModTime:    now,
		Typeflag:   tar.TypeReg,
		AccessTime: now,
		ChangeTime: now,
	}

	buildCtx = archive.ReplaceFileTarWrapper(buildCtx, map[string]archive.TarModifierFunc{
		// Overwrite docker file
		relDockerfile: func(_ string, h *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			return hdrTmpl, file, nil
		},
	})
	return buildCtx, nil
}

// CreateTempDockerfile creates a new temporary dockerfile that appends a new entrypoint and cmd
func CreateTempDockerfile(dockerfile string, entrypointArr []string) (string, error) {
	if entrypointArr == nil || len(entrypointArr) == 0 {
		return "", errors.New("Entrypoint is empty")
	}

	// Convert to string array
	entrypoint := []string{}
	for _, str := range entrypointArr {
		if str != "" {
			entrypoint = append(entrypoint, str)
		}
	}
	if len(entrypoint) == 0 {
		return "", errors.New("Entrypoint is empty")
	}

	data, err := ioutil.ReadFile(dockerfile)
	if err != nil {
		return "", err
	}

	// Overwrite entrypoint and cmd
	newDockerfileContents := string(data)
	newDockerfileContents += "\n\nENTRYPOINT [\"" + entrypoint[0] + "\"]"
	newDockerfileContents += "\nCMD [\"" + strings.Join(entrypoint[1:], "\",\"") + "\"]"

	tmpDir, err := ioutil.TempDir("", "example")
	if err != nil {
		return "", err
	}

	tmpfn := filepath.Join(tmpDir, "Dockerfile")
	if err := ioutil.WriteFile(tmpfn, []byte(newDockerfileContents), 0666); err != nil {
		return "", err
	}

	return tmpfn, nil
}
