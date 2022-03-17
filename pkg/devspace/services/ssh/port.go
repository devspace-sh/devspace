package ssh

import (
	"fmt"
	"github.com/loft-sh/devspace/helper/util/port"
	"math/rand"
	"sync"
)

var (
	portRangeStart = 10000
	portRangeEnd   = 12000
	portMap        = map[int]bool{}
	portMutex      sync.Mutex
)

func lockPort() (int, error) {
	portMutex.Lock()
	defer portMutex.Unlock()

	var (
		available bool
		err       error
	)
	for i := 0; i < 10; i++ {
		p := rand.Intn(portRangeEnd-portRangeStart+1) + portRangeStart
		if portMap[p] {
			i--
			continue
		}

		available, err = port.IsAvailable(fmt.Sprintf(":%d", p))
		if available {
			portMap[p] = true
			return p, nil
		}
	}

	return 0, fmt.Errorf("couldn't find an open port: %v", err)
}

func releasePort(p int) {
	portMutex.Lock()
	defer portMutex.Unlock()

	delete(portMap, p)
}
