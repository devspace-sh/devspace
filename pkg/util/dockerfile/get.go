package dockerfile

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

var findExposePortsRegEx = regexp.MustCompile("^EXPOSE\\s(.*)$")

// GetPorts retrieves all the exported ports from a dockerfile
func GetPorts(filename string) ([]int, error) {
	data, err := ioutil.ReadFile(filename)
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
