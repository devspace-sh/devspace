package kubectl

import (
	"net/http"

	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"
)

// UpgraderWrapper wraps the upgrader and adds a connections array
type UpgraderWrapper struct {
	Upgrader    spdy.Upgrader
	Connections []httpstream.Connection
}

// NewConnection receives a new connection
func (uw *UpgraderWrapper) NewConnection(resp *http.Response) (httpstream.Connection, error) {
	conn, err := uw.Upgrader.NewConnection(resp)
	if err != nil {
		return nil, err
	}

	uw.Connections = append(uw.Connections, conn)

	return conn, nil
}

// Close closes all connections
func (uw *UpgraderWrapper) Close() error {
	for _, conn := range uw.Connections {
		err := conn.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetUpgraderWrapper returns an upgrade wrapper for the given config
func GetUpgraderWrapper(config *rest.Config) (http.RoundTripper, *UpgraderWrapper, error) {
	wrapper, upgradeRoundTripper, err := spdy.RoundTripperFor(config)
	if err != nil {
		return nil, nil, err
	}

	return wrapper, &UpgraderWrapper{
		Upgrader:    upgradeRoundTripper,
		Connections: make([]httpstream.Connection, 0, 1),
	}, nil
}
