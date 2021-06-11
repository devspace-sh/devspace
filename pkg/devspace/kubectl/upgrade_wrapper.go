package kubectl

import (
	"k8s.io/client-go/rest"
	"net/http"
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	restclient "k8s.io/client-go/rest"
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

func (client *client) roundTripperFor(config *rest.Config) (http.RoundTripper, clientspdy.Upgrader, error) {
	tlsConfig, err := restclient.TLSConfigFor(config)
	if err != nil {
		return nil, nil, err
	}
	proxy := http.ProxyFromEnvironment
	if config.Proxy != nil {
		proxy = config.Proxy
	}
	upgradeRoundTripper := spdy.NewRoundTripperWithConfig(spdy.RoundTripperConfig{
		TLS:                      tlsConfig,
		FollowRedirects:          true,
		RequireSameHostRedirects: false,
		Proxier:                  proxy,
		PingPeriod:               time.Second * 10,
	})
	wrapper, err := restclient.HTTPWrappersForConfig(config, upgradeRoundTripper)
	if err != nil {
		return nil, nil, err
	}
	return wrapper, upgradeRoundTripper, nil
}
