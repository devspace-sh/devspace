package helper

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
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
func CreateTempDockerfile(dockerfile string, entrypoint []string, cmd []string, target string) (string, error) {
	if entrypoint == nil && cmd == nil {
		return "", errors.New("Entrypoint & cmd are empty")
	}

	data, err := ioutil.ReadFile(dockerfile)
	if err != nil {
		return "", err
	}

	// Overwrite entrypoint and cmd
	tmpDir, err := ioutil.TempDir("", "example")
	if err != nil {
		return "", err
	}

	// add the new entrypoint
	newData, err := addNewEntrypoint(string(data), entrypoint, cmd, target)
	if err != nil {
		return "", errors.Wrap(err, "add entrypoint")
	}

	tmpfn := filepath.Join(tmpDir, "Dockerfile")
	if err := ioutil.WriteFile(tmpfn, []byte(newData), 0666); err != nil {
		return "", err
	}

	return tmpfn, nil
}

var nextFromFinder = regexp.MustCompile("(?i)\n\\s*FROM")

func addNewEntrypoint(content string, entrypoint []string, cmd []string, target string) (string, error) {
	entrypointStr := ""
	if len(entrypoint) > 0 {
		entrypointStr += "\n\nENTRYPOINT [\"" + strings.Join(entrypoint, "\",\"") + "\"]\n"
	} else if entrypoint != nil {
		entrypointStr += "\n\nENTRYPOINT []\n"
	}
	if len(cmd) > 0 {
		entrypointStr += "\n\nCMD [\"" + strings.Join(cmd, "\",\"") + "\"]\n"
	} else if cmd != nil {
		entrypointStr += "\n\nCMD []\n"
	}

	if target == "" {
		return content + entrypointStr, nil
	}

	// Find the target
	targetFinder, err := regexp.Compile(fmt.Sprintf("(?i)(^|\n)\\s*FROM\\s+([a-zA-Z0-9\\:\\@\\.\\-]+)\\s+AS\\s+%s\\s*($|\n)", target))
	if err != nil {
		return "", err
	}

	matches := targetFinder.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return "", errors.Errorf("Coulnd't find target '%s' in dockerfile", target)
	} else if len(matches) > 1 {
		return "", errors.Errorf("Multiple matches for target '%s' in dockerfile", target)
	}

	// Find the next FROM statement
	nextFrom := nextFromFinder.FindStringIndex(content[matches[0][1]:])
	if len(nextFrom) != 2 {
		return content + entrypointStr, nil
	}

	return content[:matches[0][1]+nextFrom[0]] + entrypointStr + content[matches[0][1]+nextFrom[0]:], nil
}
