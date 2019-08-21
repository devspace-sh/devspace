package kubectl

import (
	"net/http"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/transport"
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

	// This is a fix to prevent the connection of getting idle and killed by the kubernetes
	// api server, this is used for sync, port forwarding and the terminal
	newConn, ok := conn.(*transport.Connection)
	if ok && newConn != nil {
		go func() {
			if newConn.Conn != nil {
				for {
					select {
					case <-newConn.Conn.CloseChan():
						return
					case <-time.After(time.Second * 10):
						newConn.Conn.Ping()
					}
				}
			}
		}()
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
	wrapper, upgradeRoundTripper, err := transport.RoundTripperFor(config)
	if err != nil {
		return nil, nil, err
	}

	return wrapper, &UpgraderWrapper{
		Upgrader:    upgradeRoundTripper,
		Connections: make([]httpstream.Connection, 0, 1),
	}, nil
}
