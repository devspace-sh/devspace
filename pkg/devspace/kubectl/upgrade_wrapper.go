package kubectl

import (
	"net/http"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/httpstream"
	clientspdy "k8s.io/client-go/transport/spdy"
)

// UpgraderWrapper wraps the upgrader and adds a connections array
type UpgraderWrapper interface {
	NewConnection(resp *http.Response) (httpstream.Connection, error)
	Close() error
}

type upgraderWrapper struct {
	Upgrader    clientspdy.Upgrader
	Connections []httpstream.Connection
}

// NewConnection receives a new connection
func (uw *upgraderWrapper) NewConnection(resp *http.Response) (httpstream.Connection, error) {
	conn, err := uw.Upgrader.NewConnection(resp)
	if err != nil {
		return nil, err
	}

	uw.Connections = append(uw.Connections, conn)
	return conn, nil
}

// Close closes all connections
func (uw *upgraderWrapper) Close() error {
	errs := []error{}
	for _, conn := range uw.Connections {
		err := conn.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

// GetUpgraderWrapper returns an upgrade wrapper for the given config @Factory
func (client *client) GetUpgraderWrapper() (http.RoundTripper, UpgraderWrapper, error) {
	wrapper, upgradeRoundTripper, err := clientspdy.RoundTripperFor(client.restConfig)
	if err != nil {
		return nil, nil, err
	}

	return wrapper, &upgraderWrapper{
		Upgrader:    upgradeRoundTripper,
		Connections: make([]httpstream.Connection, 0, 1),
	}, nil
}
