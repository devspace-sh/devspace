package util

import (
	"context"
	"io"
	"net"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

// NewClientConnection creates a new client connection for the given reader and writer
func NewClientConnection(reader io.Reader, writer io.Writer) (*grpc.ClientConn, error) {
	pipe := NewStdStreamJoint(reader, writer, false)
	
	resolver.SetDefaultScheme("passthrough")
	// Set up a connection to the server.
	return grpc.NewClient("",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return pipe, nil
		}),
		grpc.WithLocalDNSResolution(),
		grpc.WithDefaultCallOptions())
}
