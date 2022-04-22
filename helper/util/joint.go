package util

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// StdinAddr is the struct for the stdi
type StdinAddr struct {
	s string
}

// NewStdinAddr creates a new StdinAddr
func NewStdinAddr(s string) *StdinAddr {
	return &StdinAddr{s}
}

// Network implements interface
func (a *StdinAddr) Network() string {
	return "stdio"
}

func (a *StdinAddr) String() string {
	return a.s
}

// StdStreamJoint is the struct that implements the net.Conn interface
type StdStreamJoint struct {
	in     io.Reader
	out    io.Writer
	local  *StdinAddr
	remote *StdinAddr

	exitOnClose bool
}

// NewStdStreamJoint is used to implement the connection interface so we can connect to the rpc server
func NewStdStreamJoint(in io.Reader, out io.Writer, exitOnClose bool) *StdStreamJoint {
	return &StdStreamJoint{
		local:       NewStdinAddr("local"),
		remote:      NewStdinAddr("remote"),
		in:          in,
		out:         out,
		exitOnClose: exitOnClose,
	}
}

// LocalAddr implements interface
func (s *StdStreamJoint) LocalAddr() net.Addr {
	return s.local
}

// RemoteAddr implements interface
func (s *StdStreamJoint) RemoteAddr() net.Addr {
	return s.remote
}

// Read implements interface
func (s *StdStreamJoint) Read(b []byte) (n int, err error) {
	return s.in.Read(b)
}

// Write implements interface
func (s *StdStreamJoint) Write(b []byte) (n int, err error) {
	return s.out.Write(b)
}

// Close implements interface
func (s *StdStreamJoint) Close() error {
	if s.exitOnClose {
		// We kill ourself here because the streams are closed
		_, _ = fmt.Fprintf(os.Stderr, "Streams are closed")
		os.Exit(1)
	}

	return nil
}

// SetDeadline implements interface
func (s *StdStreamJoint) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline implements interface
func (s *StdStreamJoint) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline implements interface
func (s *StdStreamJoint) SetWriteDeadline(t time.Time) error {
	return nil
}
