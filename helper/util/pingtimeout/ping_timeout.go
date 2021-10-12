package pingtimeout

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"sync"
	"time"
)

const (
	pingTimeout = time.Second * 60
)

type PingTimeout struct {
	lastPing      *time.Time
	lastPingMutex sync.Mutex
	lastPingOnce  sync.Once
}

func (p *PingTimeout) Ping() {
	p.lastPingMutex.Lock()
	defer p.lastPingMutex.Unlock()

	now := time.Now()
	p.lastPing = &now
}

func (p *PingTimeout) Start(stopChan chan struct{}) {
	p.lastPingOnce.Do(func() {
		p.Ping()
		go func() {
			wait.Until(func() {
				p.lastPingMutex.Lock()
				defer p.lastPingMutex.Unlock()

				if p.lastPing == nil {
					return
				}

				if time.Now().After(p.lastPing.Add(pingTimeout)) {
					_, _ = fmt.Fprintf(os.Stderr, "Pings timed out")
					os.Exit(1)
				}
			}, time.Second, stopChan)
		}()
	})
}
