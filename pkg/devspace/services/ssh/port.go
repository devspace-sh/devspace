package ssh

import (
	"fmt"
	"github.com/loft-sh/devspace/helper/util/port"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"math/rand"
	"path/filepath"
	"sync"
)

type PortManager interface {
	LockPort() (int, error)
	LockSpecificPort(p int) error
	ReleasePort(p int)
}

var (
	portManager     PortManager
	portManagerOnce sync.Once
)

func GetInstance(log log.Logger) PortManager {
	portManagerOnce.Do(func() {
		portManager = NewManager(log)
	})
	return portManager
}

func NewManager(log log.Logger) PortManager {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Errorf("%v", errors.Wrap(err, "get home dir"))
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	hosts, err := ParseDevSpaceHosts(sshConfigPath)
	if err != nil {
		log.Errorf("error parsing %s: %v", sshConfigPath, err)
	}
	reservedPorts := map[int]bool{}
	for _, h := range hosts {
		reservedPorts[h.Port] = true
	}
	sshConfigPath = filepath.Join(homeDir, ".ssh", "devspace_config")
	hosts, err = ParseDevSpaceHosts(sshConfigPath)
	if err != nil {
		log.Errorf("error parsing %s: %v", sshConfigPath, err)
	}
	for _, h := range hosts {
		reservedPorts[h.Port] = true
	}

	return &manager{
		reservedPorts:  reservedPorts,
		portRangeStart: 10000,
		portRangeEnd:   12000,
		portMap:        map[int]bool{},
	}
}

type manager struct {
	m sync.Mutex

	reservedPorts map[int]bool

	portRangeStart int
	portRangeEnd   int
	portMap        map[int]bool
}

func (m *manager) LockSpecificPort(p int) error {
	m.m.Lock()
	defer m.m.Unlock()

	if m.portMap[p] {
		return fmt.Errorf("port %d already in use", p)
	}

	available, err := port.IsAvailable(fmt.Sprintf(":%d", p))
	if err != nil {
		return err
	} else if !available {
		return fmt.Errorf("port %d is already in use %v", p, err)
	}

	m.portMap[p] = true
	return nil
}

func (m *manager) LockPort() (int, error) {
	m.m.Lock()
	defer m.m.Unlock()

	var (
		available bool
		err       error
	)
	for i := 0; i < 10; i++ {
		p := rand.Intn(m.portRangeEnd-m.portRangeStart+1) + m.portRangeStart
		if m.portMap[p] || m.reservedPorts[p] {
			i--
			continue
		}

		available, err = port.IsAvailable(fmt.Sprintf(":%d", p))
		if available {
			m.portMap[p] = true
			return p, nil
		}
	}

	return 0, fmt.Errorf("couldn't find an open port: %v", err)
}

func (m *manager) ReleasePort(p int) {
	m.m.Lock()
	defer m.m.Unlock()

	delete(m.portMap, p)
}
