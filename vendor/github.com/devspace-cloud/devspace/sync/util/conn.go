package util

import (
	"io"
	"net"
	"time"

	"google.golang.org/grpc"
)

// NewClientConnection creates a new client connection for the given reader and writer
func NewClientConnection(reader io.Reader, writer io.Writer) (*grpc.ClientConn, error) {
	pipe := NewStdStreamJoint(reader, writer)

	// Set up a connection to the server.
	return grpc.Dial("", grpc.WithInsecure(), grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
		return pipe, nil
	}))
}
