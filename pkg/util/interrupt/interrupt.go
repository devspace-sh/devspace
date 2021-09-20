/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package interrupt

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/google/uuid"
)

// Global is the global interrupt handler
var Global = New(func(o os.Signal) {
	os.Exit(1)
})

// terminationSignals are signals that cause the program to exit in the
// supported platforms (linux, darwin, windows).
var terminationSignals = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

// Handler guarantees execution of notifications after a critical section (the function passed
// to a Run method), even in the presence of process termination. It guarantees exactly once
// invocation of the provided notify functions.
type Handler struct {
	notifyMutex sync.Mutex
	notify      []notify
	final       func(os.Signal)
	once        sync.Once

	channelMutex sync.Mutex
	channel      chan os.Signal
}

type notify struct {
	id string
	fn func()
}

// New creates a new handler that guarantees all notify functions are run after the critical
// section exits (or is interrupted by the OS), then invokes the final handler. If no final
// handler is specified, the default final is `os.Exit(1)`. A handler can only be used for
// one critical section.
func New(final func(os.Signal)) *Handler {
	return &Handler{
		final:  final,
		notify: []notify{},
	}
}

// Signal is called when an os.Signal is received, and guarantees that all notifications
// are executed, then the final handler is executed. This function should only be called once
// per Handler instance.
func (h *Handler) Signal(s os.Signal) {
	h.once.Do(func() {
		h.notifyMutex.Lock()
		defer h.notifyMutex.Unlock()
		for _, fn := range h.notify {
			fn.fn()
		}
		if h.final == nil {
			os.Exit(1)
		}
		h.final(s)
	})
}

func (h *Handler) register(id string, fn func()) {
	h.notifyMutex.Lock()
	defer h.notifyMutex.Unlock()

	h.notify = append(h.notify, notify{
		id: id,
		fn: fn,
	})
}

func (h *Handler) unregister(id string) {
	h.notifyMutex.Lock()
	defer h.notifyMutex.Unlock()

	newNotify := []notify{}
	for _, n := range h.notify {
		if id == n.id {
			continue
		}

		newNotify = append(newNotify, n)
	}
	h.notify = newNotify
}

// Run ensures that the function fn is run and runs the notify function if interrupted meanwhile
func (h *Handler) Run(fn func() error, notify func()) error {
	id := uuid.New().String()
	h.register(id, notify)
	defer h.unregister(id)

	return fn()
}

// RunAlways ensures that the function fn is run and runs the notify function if interrupted meanwhile or the function has ended
func (h *Handler) RunAlways(fn func() error, notify func()) error {
	defer notify()

	id := uuid.New().String()
	h.register(id, notify)
	defer h.unregister(id)

	return fn()
}

// Start ensures the handler is started and ready for incoming signals
func (h *Handler) Start() {
	h.channelMutex.Lock()
	defer h.channelMutex.Unlock()

	if h.channel != nil {
		return
	}

	h.channel = make(chan os.Signal, 1)
	signal.Notify(h.channel, terminationSignals...)
	go func(channel chan os.Signal) {
		sig, ok := <-channel
		if !ok {
			return
		}
		h.Signal(sig)
	}(h.channel)
}

// Stop ensures we do not watch for incoming signals anymore
func (h *Handler) Stop() {
	h.channelMutex.Lock()
	defer h.channelMutex.Unlock()

	if h.channel != nil {
		signal.Stop(h.channel)
		close(h.channel)
		h.channel = nil
	}
}
