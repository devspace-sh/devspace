package registry

import "context"

type ExcludePayload struct {
	// RunID is the run id of the process that owns the dependency
	RunID string `json:"runID,omitempty"`

	// DependencyName is the name of the dependency to exclude
	DependencyName string `json:"dependencyName,omitempty"`
}

type PingPayload struct {
	// RunID is the run id of the process that owns the dependency
	RunID string `json:"runID,omitempty"`
}

type InterProcess interface {
	// Ping pings the remote server to check if it still exists. Returns true if
	// successful and false if unsuccessful and an error if the server is not reachable
	Ping(ctx context.Context, server string, payload *PingPayload) (bool, error)

	// ExcludeDependency tells the remote server to exclude a certain dependency
	ExcludeDependency(ctx context.Context, server string, excludePayload *ExcludePayload) error
}

func NewInterProcessCommunicator() InterProcess {
	return &dummyImplementation{}
}

type dummyImplementation struct{}

func (d *dummyImplementation) Ping(ctx context.Context, server string, pingPayload *PingPayload) (bool, error) {
	return false, nil
}

func (d *dummyImplementation) ExcludeDependency(ctx context.Context, server string, excludePayload *ExcludePayload) error {
	return nil
}
