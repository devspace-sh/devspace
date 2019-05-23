package util

import (
	"fmt"
	"net"
)

// NewStdinListener creates a new stdin listener
func NewStdinListener() *StdinListener {
	return &StdinListener{
		connChan: make(chan net.Conn),
	}
}

// StdinListener implements the listener interface
type StdinListener struct {
	closed   bool
	connChan chan net.Conn
}

// Ready implements interface
func (lis *StdinListener) Ready(conn net.Conn) {
	lis.connChan <- conn
}

// Accept implements interface
func (lis *StdinListener) Accept() (net.Conn, error) {
	conn := <-lis.connChan
	fmt.Println("Connection accepted")

	return conn, nil
}

// Close implements interface
func (lis *StdinListener) Close() error {
	lis.closed = true
	return nil
}

// Addr implements interface
func (lis *StdinListener) Addr() net.Addr {
	return NewStdinAddr("listener")
}
