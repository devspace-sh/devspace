package tunnel

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/portforward"
	"github.com/mgutz/ansi"
	"io"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/tunnel"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func ReceiveData(stream remote.Tunnel_InitTunnelClient, closeStream <-chan bool, sessionsOut chan<- *tunnel.Session, port int32, scheme string, log logpkg.Logger) error {
loop:
	for {
		m, err := stream.Recv()
		select {
		case <-closeStream:
			log.Debugf("closing listener on %d", port)
			_ = stream.CloseSend()
			break loop
		case <-stream.Context().Done():
			_ = stream.CloseSend()
			break loop
		default:
			if err != nil {
				return fmt.Errorf("error reading from stream: %v", err)
			} else if m.HasErr {
				_ = stream.CloseSend()
				if m.LogMessage == nil {
					return fmt.Errorf("remote error: unknown")
				}

				return fmt.Errorf("helper error: %s", m.LogMessage.Message)
			}

			requestID, err := uuid.Parse(m.RequestId)
			if err != nil {
				log.Errorf("%s; failed parsing session uuid from stream, skipping", m.RequestId)
				continue
			}

			session, exists := tunnel.GetSession(requestID)
			if !exists {
				log.Debugf("new connection %s", requestID)

				// new session
				conn, err := net.DialTimeout(strings.ToLower(scheme), fmt.Sprintf("localhost:%d", port), time.Millisecond*500)
				if err != nil {
					log.Errorf("failed connecting to localhost on port %d scheme %s: %v", port, scheme, err)
					// close the remote connection
					resp := &remote.SocketDataRequest{
						RequestId:   requestID.String(),
						ShouldClose: true,
					}
					err := stream.Send(resp)
					if err != nil {
						log.Errorf("failed sending close message to tunnel stream: %v", err)
					}

					continue
				}

				session, err = tunnel.NewSessionFromStream(requestID, conn)
				if err != nil {
					log.Errorf("%s; error creating new session from stream: %v", m.RequestId, err)
					continue
				}

				go ReadFromSession(session, sessionsOut, log)
			} else if m.ShouldClose {
				session.Open = false
			}

			// process the data from the server
			handleStreamData(m, session, log)
		}
	}
	return nil
}

func handleStreamData(m *remote.SocketDataResponse, session *tunnel.Session, log logpkg.Logger) {
	if !session.Open {
		session.Close()
		return
	}

	data := m.GetData()
	log.Debugf("received %d bytes from server", len(data))
	if len(data) > 0 {
		session.Lock()
		_, err := session.Conn.Write(data)
		session.Unlock()
		log.Debugf("wrote %d bytes to conn", len(data))
		if err != nil {
			log.Warnf("%s: failed writing to socket, closing session: %v", session.ID.String(), err)
			session.Close()
			return
		}
	}
}

func ReadFromSession(session *tunnel.Session, sessionsOut chan<- *tunnel.Session, log logpkg.Logger) {
	log.Debugf("started reading conn %s", session.ID)
	defer log.Debugf("finished reading conn %s", session.ID)

	conn := session.Conn
	buff := make([]byte, tunnel.BufferSize)

loop:
	for {
		br, err := conn.Read(buff)
		select {
		case <-session.Context.Done():
			return
		default:
			if err != nil {
				if err != io.EOF {
					log.Errorf("%s: failed reading from socket, exiting: %v", session.ID.String(), err)
				} else {
					log.Debugf("read EOF from conn")
				}
				session.Open = false
				sessionsOut <- session
				break loop
			}

			log.Debugf("read %d bytes from conn", br)
			if br > 0 {
				session.Lock()
				_, err = session.Buf.Write(buff[0:br])
				session.Unlock()
				log.Debugf("wrote %d bytes to session", br)
			}
			if err != nil {
				log.Errorf("%s: failed writing to session buffer: %v", session.ID, err)
				break loop
			}

			sessionsOut <- session
		}
	}
}

func SendData(stream remote.Tunnel_InitTunnelClient, sessions <-chan *tunnel.Session, closeChan <-chan bool, log logpkg.Logger) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-closeChan:
			return nil
		case session := <-sessions:
			// read the bytes from the buffer
			// but allow it to keep growing while we send the response
			session.Lock()
			bys := session.Buf.Len()
			bytes := make([]byte, bys)
			_, err := session.Buf.Read(bytes)
			if err != nil {
				session.Unlock()
				return fmt.Errorf("failed reading stream from session %v, exiting", err)
			}
			log.Debugf("read %d from buffer out of %d available", len(bytes), bys)
			resp := &remote.SocketDataRequest{
				RequestId:   session.ID.String(),
				Data:        bytes,
				ShouldClose: !session.Open,
			}
			session.Unlock()

			log.Debugf("sending %d bytes to server", len(bytes))
			err = stream.Send(resp)
			if err != nil {
				return fmt.Errorf("failed sending message to tunnel stream, exiting")
			}
			log.Debugf("sent %d bytes to server", len(bytes))
		}
	}
}

func StartReverseForward(ctx context.Context, reader io.ReadCloser, writer io.WriteCloser, tunnels []*latest.PortMapping, stopChan chan struct{}, namespace string, name string, log logpkg.Logger) error {
	scheme := "TCP"
	closeStreams := make([]chan bool, len(tunnels))
	defer func() {
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

	errorsChan := make(chan error, 2*len(tunnels)+1)
	closeStream := make(chan struct{})
	defer close(closeStream)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-closeStream:
				return
			case <-stopChan:
				return
			case <-time.After(time.Second * 20):
				ctx, cancel := context.WithTimeout(ctx, time.Second*20)
				_, err := client.Ping(ctx, &remote.Empty{})
				cancel()
				if err != nil {
					errorsChan <- errors.Wrap(err, "ping connection")
					return
				}
			}
		}
	}()

	for i, portMapping := range tunnels {
		if portMapping.Port == "" {
			return fmt.Errorf("local port cannot be undefined")
		}

		mappings, err := portforward.ParsePorts([]string{portMapping.Port})
		if err != nil {
			return fmt.Errorf("error parsing port %s: %v", portMapping.Port, err)
		}

		localPort := mappings[0].Local
		remotePort := mappings[0].Remote
		c := make(chan bool, 1)
		go func(closeStream chan bool, localPort, remotePort int32) {
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
				err = SendData(stream, sessions, closeStream, logFile)
				if err != nil {
					errorsChan <- err
				}
			}()

			// wait until close
			log.Donef("Port forwarding started on: %s", ansi.Color(fmt.Sprintf("%d <- %d", localPort, remotePort), "white+b"))
			<-closeStream
		}(c, int32(localPort), int32(remotePort))
		closeStreams[i] = c
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errorsChan:
		return err
	case <-stopChan:
		return nil
	}
}
