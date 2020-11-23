package tunnel

import (
	"fmt"
	"github.com/devspace-cloud/devspace/helper/remote"
	"github.com/devspace-cloud/devspace/helper/tunnel"
	"github.com/devspace-cloud/devspace/helper/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	logpkg "github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"io"
	"net"
	"strings"
	"time"
)

const (
	bufferSize = 1024 * 32
)

type Message struct {
	c *net.Conn
	d *[]byte
}

func ReceiveData(stream remote.Tunnel_InitTunnelClient, closeStream <-chan bool, sessionsOut chan<- *tunnel.Session, port int32, scheme string, log logpkg.Logger) error {
loop:
	for {
		select {
		case <-closeStream:
			_ = stream.CloseSend()
			break loop
		default:
			m, err := stream.Recv()
			if err != nil {
				return fmt.Errorf("error reading from stream: %v", err)
			}
			requestId, err := uuid.Parse(m.RequestId)
			if err != nil {
				log.Errorf("%s; failed parsing session uuid from stream, skipping", m.RequestId)
				continue
			}
			session, exists := tunnel.GetSession(requestId)
			if exists == false {
				if m.ShouldClose == false {
					// new session
					conn, err := net.DialTimeout(strings.ToLower(scheme), fmt.Sprintf("localhost:%d", port), time.Millisecond*500)
					if err != nil {
						log.Errorf("failed connecting to localhost on port %d scheme %s: %v", port, scheme, err)
						continue
					}
					session = tunnel.NewSessionFromStream(requestId, conn)
					go ReadFromSession(session, sessionsOut, log)
				} else {
					session = tunnel.NewSessionFromStream(requestId, nil)
					session.Open = false
				}
			}
			handleStreamData(m, session, log)
		}
	}

	return nil
}

func handleStreamData(m *remote.SocketDataResponse, session *tunnel.Session, log logpkg.Logger) {
	if session.Open == false {
		if session.Conn != nil {
			ok, err := tunnel.CloseSession(session.Id)
			if ok != true {
				log.Warnf("%s: failed closing session: %v", session.Id.String(), err)
			}
		}
	} else {
		c := session.Conn
		data := m.GetData()
		if len(data) > 0 {
			session.Lock.Lock()
			_, err := c.Write(data)
			session.Lock.Unlock()
			if err != nil {
				log.Warnf("%s: failed writing to socket, closing session: %v", session.Id.String(), err)
				ok, err := tunnel.CloseSession(session.Id)
				if ok != true {
					log.Warnf("%s: failed closing session: %v", session.Id.String(), err)
				}
			}
		}
	}
}

func ReadFromSession(session *tunnel.Session, sessionsOut chan<- *tunnel.Session, log logpkg.Logger) {
	conn := session.Conn
	buff := make([]byte, bufferSize)
	for {
		br, err := conn.Read(buff)
		if err != nil {
			if err != io.EOF {
				log.Errorf("%s: failed reading from socket, exiting: %v", session.Id.String(), err)
			}
			session.Open = false
			sessionsOut <- session
			break
		}
		if br > 0 {
			session.Lock.Lock()
			_, err = session.Buf.Write(buff[:br])
			session.Lock.Unlock()
			if br == len(buff) {
				newSize := len(buff) * 2
				buff = make([]byte, newSize)
			}
		}
		if err != nil {
			log.Errorf("%s: failed writing to session buffer: %v", session.Id, err)
			break
		}
		sessionsOut <- session
	}
}

func SendData(stream remote.Tunnel_InitTunnelClient, sessions <-chan *tunnel.Session, closeChan <-chan bool) error {
	errorChan := make(chan error, 10)
	for {
		select {
		case err := <-errorChan:
			return err
		case <-closeChan:
			return nil
		case session := <-sessions:
			session.Lock.Lock()
			bys := session.Buf.Len()
			bytes := make([]byte, bys)
			_, _ = session.Buf.Read(bytes)

			resp := &remote.SocketDataRequest{
				RequestId:   session.Id.String(),
				Data:        bytes,
				ShouldClose: false,
			}
			if session.Open == false {
				resp.ShouldClose = true
			}
			session.Lock.Unlock()
			err := stream.Send(resp)
			if err != nil {
				errorChan <- fmt.Errorf("failed sending message to tunnel stream, exiting; %v", err)
			}
		}
	}
}

func StartReverseForward(reader io.ReadCloser, writer io.WriteCloser, tunnels []*latest.PortMapping, stopChan chan error, log logpkg.Logger) error {
	scheme := "TCP"
	closeStreams := make([]chan bool, len(tunnels))
	go func() {
		for _, c := range closeStreams {
			if c == nil {
				continue
			}

			close(c)
		}
	}()

	// Create client
	conn, err := util.NewClientConnection(reader, writer)
	if err != nil {
		return errors.Wrap(err, "new client connection")
	}
	client := remote.NewTunnelClient(conn)
	logFile := logpkg.GetFileLogger("reverse-portforwarding")

	errorsChan := make(chan error, 2*len(tunnels))
	for i, portMapping := range tunnels {
		if portMapping.LocalPort == nil {
			return fmt.Errorf("local port cannot be undefined")
		}

		localPort := *portMapping.LocalPort
		remotePort := localPort
		if portMapping.RemotePort != nil {
			remotePort = *portMapping.RemotePort
		}

		c := make(chan bool, 1)
		go func(closeStream chan bool, localPort, remotePort int32) {
			ctx := context.Background()
			tunnelScheme, ok := remote.TunnelScheme_value[scheme]
			if !ok {
				errorsChan <- fmt.Errorf("unsupported connection scheme %s", scheme)
				return
			}
			req := &remote.SocketDataRequest{
				Port:     remotePort,
				LogLevel: 0,
				Scheme:   remote.TunnelScheme(tunnelScheme),
			}
			stream, err := client.InitTunnel(ctx)
			if err != nil {
				errorsChan <- fmt.Errorf("error sending init tunnel request: %v", err)
				return
			}

			err = stream.Send(req)
			if err != nil {
				errorsChan <- fmt.Errorf("failed to send initial tunnel request to server")
				return
			}

			sessions := make(chan *tunnel.Session)
			go func() {
				err = ReceiveData(stream, closeStream, sessions, localPort, scheme, logFile)
				if err != nil {
					errorsChan <- err
				}
			}()
			go func() {
				err = SendData(stream, sessions, closeStream)
				if err != nil {
					errorsChan <- err
				}
			}()

			// wait until close
			log.Donef("Reverse port forwarding started at %d:%d", remotePort, localPort)
			<-closeStream
		}(c, int32(localPort), int32(remotePort))
		closeStreams[i] = c
	}

	select {
	case err := <-errorsChan:
		return err
	case <-stopChan:
		return nil
	}
}
