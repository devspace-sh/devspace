package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

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
	ExcludeDependency(ctx context.Context, server string, excludePayload *ExcludePayload) (bool, error)
}

func NewInterProcessCommunicator() InterProcess {
	return &requester{}
}

type requester struct{}

func (d *requester) Ping(ctx context.Context, server string, pingPayload *PingPayload) (bool, error) {
	out, err := json.Marshal(pingPayload)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", server+"/api/ping", bytes.NewReader(out))
	if err != nil {
		return false, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func (d *requester) ExcludeDependency(ctx context.Context, server string, excludePayload *ExcludePayload) (bool, error) {
	out, err := json.Marshal(excludePayload)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", server+"/api/exclude-dependency", bytes.NewReader(out))
	if err != nil {
		return false, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden {
		return false, nil
	}
	return true, nil
}
