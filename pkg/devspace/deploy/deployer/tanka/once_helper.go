package tanka

import (
	"fmt"
	"sync"
)

var (
	mutex   sync.Mutex
	onceMap map[string]*sync.Once
)

func init() {
	onceMap = make(map[string]*sync.Once)
}

func GetOnce(operation, path string) *sync.Once {
	var once *sync.Once
	var ok bool

	mutex.Lock()
	defer mutex.Unlock()

	if once, ok = onceMap[fmt.Sprintf("%s-%s", operation, path)]; !ok {
		once = &sync.Once{}
		onceMap[fmt.Sprintf("%s-%s", operation, path)] = once
	}
	return once
}
