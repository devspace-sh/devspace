package dockerfile

import (
	"bytes"
	"github.com/docker/distribution/reference"
	dockerregistry "github.com/docker/docker/registry"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var findExposePortsRegEx = regexp.MustCompile(`^EXPOSE\s(.*)$`)

// GetStrippedDockerImageName returns a tag stripped image name and checks if it's a valid image name
func GetStrippedDockerImageName(imageName string) (string, string, error) {
	imageName = strings.TrimSpace(imageName)

	// Check if we can parse the name
	ref, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return "", "", err
	}

	// Check if there was a tag
	tag := ""
	if refTagged, ok := ref.(reference.NamedTagged); ok {
		tag = refTagged.Tag()
	}

	repoInfo, err := dockerregistry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", "", err
	}

	if repoInfo.Index.Official {
		// strip docker.io and library from image
		return strings.TrimPrefix(strings.TrimPrefix(reference.TrimNamed(ref).Name(), repoInfo.Index.Name+"/library/"), repoInfo.Index.Name+"/"), tag, nil
	}

	return reference.TrimNamed(ref).Name(), tag, nil
}

// GetPorts retrieves all the exported ports from a dockerfile
func GetPorts(filename string) ([]int, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	data = NormalizeNewlines(data)
	lines := strings.Split(string(data), "\n")
	ports := []int{}

	for _, line := range lines {
		match := findExposePortsRegEx.FindStringSubmatch(line)
		if match == nil || len(match) != 2 {
			continue
		}

		portStrings := strings.Split(match[1], " ")

	OUTER:
		for _, port := range portStrings {
			if port == "" {
				continue
			}

			intPort, err := strconv.Atoi(strings.Split(port, "/")[0])
			if err != nil {
				return nil, err
			}

			// Check if port already exists
			for _, existingPort := range ports {
				if existingPort == intPort {
					continue OUTER
				}
			}

			ports = append(ports, intPort)
		}
	}

	return ports, nil
}

// NormalizeNewlines normalizes \r\n (windows) and \r (mac)
// into \n (unix)
func NormalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.Replace(d, []byte{13, 10}, []byte{10}, -1)
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.Replace(d, []byte{13}, []byte{10}, -1)
	return d
}
