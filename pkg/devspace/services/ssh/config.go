package ssh

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var configLock sync.Mutex

var (
	MarkerStartPrefix = "# DevSpace Start "
	MarkerEndPrefix   = "# DevSpace End "
)

func configureSSHConfig(host, port string, useInclude bool, log log.Logger) error {
	if useInclude {
		return configureSSHConfigSeparateFile(host, port, log)
	}

	return configureSSHConfigSameFile(host, port, log)
}

func configureSSHConfigSameFile(host, port string, log log.Logger) error {
	configLock.Lock()
	defer configLock.Unlock()

	homeDir, err := homedir.Dir()
	if err != nil {
		return errors.Wrap(err, "get home dir")
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	newFile, err := addHost(sshConfigPath, host, port)
	if err != nil {
		return errors.Wrap(err, "parse ssh config")
	}

	err = os.MkdirAll(filepath.Dir(sshConfigPath), 0755)
	if err != nil {
		log.Debugf("error creating ssh directory: %v", err)
	}

	err = os.WriteFile(sshConfigPath, []byte(newFile), 0600)
	if err != nil {
		return errors.Wrap(err, "write ssh config")
	}

	return nil
}

func configureSSHConfigSeparateFile(host, port string, log log.Logger) error {
	configLock.Lock()
	defer configLock.Unlock()

	homeDir, err := homedir.Dir()
	if err != nil {
		return errors.Wrap(err, "get home dir")
	}

	devSpaceSSHConfigPath := filepath.Join(homeDir, ".ssh", "devspace_config")
	newFile, err := addHost(devSpaceSSHConfigPath, host, port)
	if err != nil {
		return errors.Wrap(err, "parse devspace ssh config")
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	newSSHFile, err := includeDevSpaceConfig(sshConfigPath)
	if err != nil {
		return errors.Wrap(err, "parse ssh config")
	}

	err = os.MkdirAll(filepath.Dir(sshConfigPath), 0755)
	if err != nil {
		log.Debugf("error creating ssh directory: %v", err)
	}

	if newSSHFile != "" {
		err = os.WriteFile(sshConfigPath, []byte(newSSHFile), 0600)
		if err != nil {
			return errors.Wrap(err, "write ssh config")
		}
	}

	err = os.WriteFile(devSpaceSSHConfigPath, []byte(newFile), 0600)
	if err != nil {
		return errors.Wrap(err, "write devspace ssh config")
	}

	return nil
}

type DevSpaceSSHEntry struct {
	Host     string
	Hostname string
	Port     int
}

func ParseDevSpaceHosts(path string) ([]DevSpaceSSHEntry, error) {
	var reader io.Reader
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		reader = strings.NewReader("")
	} else {
		reader = f
		defer f.Close()
	}

	configScanner := scanner.NewScanner(reader)
	inSection := false

	entries := []DevSpaceSSHEntry{}
	current := &DevSpaceSSHEntry{}
	for configScanner.Scan() {
		text := strings.TrimSpace(configScanner.Text())
		if strings.HasPrefix(text, MarkerStartPrefix) {
			inSection = true
		} else if strings.HasPrefix(text, MarkerEndPrefix) {
			if current.Host != "" && current.Port > 0 && current.Hostname != "" {
				entries = append(entries, *current)
			}
			current = &DevSpaceSSHEntry{}
			inSection = false
		} else if inSection {
			if strings.HasPrefix(text, "Host ") {
				current.Host = strings.TrimPrefix(text, "Host ")
			}
			if strings.HasPrefix(text, "Port ") {
				port := strings.TrimPrefix(text, "Port ")
				intPort, err := strconv.Atoi(port)
				if err == nil {
					current.Port = intPort
				}
			}
			if strings.HasPrefix(text, "HostName ") {
				current.Hostname = strings.TrimPrefix(text, "HostName ")
			}
		}
	}
	if configScanner.Err() != nil {
		return nil, errors.Wrap(err, "parse ssh config")
	}

	return entries, nil
}

func includeDevSpaceConfig(path string) (string, error) {
	var reader io.Reader
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		reader = strings.NewReader("")
	} else {
		reader = f
		defer f.Close()
	}

	configScanner := scanner.NewScanner(reader)
	newLines := []string{}
	startMarker := "# DevSpace Start"
	for configScanner.Scan() {
		text := configScanner.Text()
		if strings.HasPrefix(text, startMarker) {
			return "", nil
		}

		newLines = append(newLines, text)
	}
	if configScanner.Err() != nil {
		return "", errors.Wrap(err, "parse ssh config")
	}

	// add new section
	newLines = append(newLines, startMarker)
	newLines = append(newLines, "Match all")
	newLines = append(newLines, "Include devspace_config")
	newLines = append(newLines, "# DevSpace End")
	return strings.Join(newLines, "\n"), nil
}

func addHost(path, host, port string) (string, error) {
	var reader io.Reader
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}

		reader = strings.NewReader("")
	} else {
		reader = f
		defer f.Close()
	}

	configScanner := scanner.NewScanner(reader)
	newLines := []string{}
	inSection := false
	startMarker := "# DevSpace Start " + host
	endMarker := "# DevSpace End " + host
	for configScanner.Scan() {
		text := configScanner.Text()
		if strings.HasPrefix(text, startMarker) {
			inSection = true
		} else if strings.HasPrefix(text, endMarker) {
			inSection = false
		} else if !inSection {
			newLines = append(newLines, text)
		}
	}
	if configScanner.Err() != nil {
		return "", errors.Wrap(err, "parse ssh config")
	}

	// add new section
	newLines = append(newLines, startMarker)
	newLines = append(newLines, "Host "+host)
	newLines = append(newLines, "  HostName localhost")
	newLines = append(newLines, "  LogLevel error")
	newLines = append(newLines, "  Port "+port)
	newLines = append(newLines, "  IdentityFile \""+DevSpaceSSHPrivateKeyFile+"\"")
	newLines = append(newLines, "  StrictHostKeyChecking no")
	newLines = append(newLines, "  UserKnownHostsFile /dev/null")
	newLines = append(newLines, "  User devspace")
	newLines = append(newLines, endMarker)
	return strings.Join(newLines, "\n"), nil
}
