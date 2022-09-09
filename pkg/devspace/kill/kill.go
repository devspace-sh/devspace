package kill

import "sync"

// stopDevSpace can be used to stop DevSpace globally
var stopDevSpace func(message string)
var stopDevSpaceMutex sync.Mutex

func StopDevSpace(message string) {
	stopDevSpaceMutex.Lock()
	defer stopDevSpaceMutex.Unlock()

	// don't block here
	go stopDevSpace(message)
}

func SetStopFunction(stopFn func(message string)) {
	stopDevSpaceMutex.Lock()
	defer stopDevSpaceMutex.Unlock()

	stopDevSpace = stopFn
}
