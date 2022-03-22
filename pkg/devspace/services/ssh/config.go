package ssh

import (
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/scanner"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var configLock sync.Mutex

func configureSSHConfig(host, port string, log log.Logger) error {
	configLock.Lock()
	defer configLock.Unlock()

	homeDir, err := homedir.Dir()
	if err != nil {
		return errors.Wrap(err, "get home dir")
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	newFile, err := replaceHost(sshConfigPath, host, port)
	if err != nil {
		return errors.Wrap(err, "parse ssh config")
	}

	err = os.MkdirAll(filepath.Dir(sshConfigPath), 0755)
	if err != nil {
		log.Debugf("error creating ssh directory: %v", err)
	}

	err = ioutil.WriteFile(sshConfigPath, []byte(newFile), 0600)
	if err != nil {
		return errors.Wrap(err, "write ssh config")
	}

	return nil
}

func replaceHost(path, host, port string) (string, error) {
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
	newLines = append(newLines, "")
	newLines = append(newLines, startMarker)
	newLines = append(newLines, "Host "+host)
	newLines = append(newLines, "  HostName localhost")
	newLines = append(newLines, "  Port "+port)
	newLines = append(newLines, "  IdentityFile "+DevSpaceSSHPrivateKeyFile)
	newLines = append(newLines, "  StrictHostKeyChecking no")
	newLines = append(newLines, "  UserKnownHostsFile /dev/null")
	newLines = append(newLines, endMarker)
	newLines = append(newLines, "")

	return strings.Join(newLines, "\n"), nil
}
