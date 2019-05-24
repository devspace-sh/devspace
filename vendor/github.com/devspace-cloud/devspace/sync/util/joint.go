package util

import (
	"io"
	"net"
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
	closed bool
	local  *StdinAddr
	remote *StdinAddr
}

// NewStdStreamJoint is used to implement the connection interface so we can connect to the rpc server
func NewStdStreamJoint(in io.Reader, out io.Writer) *StdStreamJoint {
	return &StdStreamJoint{
		local:  NewStdinAddr("local"),
		remote: NewStdinAddr("remote"),
		in:     in,
		out:    out,
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

// Read impements interface
func (s *StdStreamJoint) Read(b []byte) (n int, err error) {
	return s.in.Read(b)
}

// Write implements interface
func (s *StdStreamJoint) Write(b []byte) (n int, err error) {
	return s.out.Write(b)
}

// Close implements interface
func (s *StdStreamJoint) Close() error {
	s.closed = true
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
