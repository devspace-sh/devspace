package port

import (
	"net"
	"strconv"
)

// CheckHostPort if a port is available
func CheckHostPort(host string, port int) (status bool, err error) {
	// Concatenate a colon and the port
	host = host + ":" + strconv.Itoa(port)

	// Try to create a server with the port
	server, err := net.Listen("tcp", host)

	// if it fails then the port is likely taken
	if err != nil {
		return false, err
	}

	// close the server
	server.Close()

	// we successfully used and closed the port
	// so it's now available to be used again
	return true, nil
}

// Check if a port is available
func Check(port int) (status bool, err error) {
	return CheckHostPort("", port)
}
