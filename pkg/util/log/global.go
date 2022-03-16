package log

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"sync"
)

var (
	globalItem  string
	globalMutex sync.Mutex
)

func AcquireGlobalSilence() (string, error) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalItem != "" {
		return "", fmt.Errorf("seems like there is already another terminal or question being asked currently")
	}

	globalItem = randutil.GenerateRandomString(12)
	return globalItem, nil
}

func ReleaseGlobalSilence(id string) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalItem == id {
		globalItem = ""
	}
}
