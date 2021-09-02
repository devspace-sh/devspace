package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
	"net"
	"os"
	"strings"
)

type tunnelServer struct{}

var debugModeEnabled = os.Getenv("DEVSPACE_HELPER_DEBUG") == "true"

func logErrorf(message string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, message+"\n", args...)
}

func logDebugf(message string, args ...interface{}) {
	if debugModeEnabled {
		_, _ = fmt.Fprintf(os.Stderr, message+"\n", args...)
	}
}

func StartTunnelServer(reader io.Reader, writer io.Writer, exitOnClose bool) error {
	pipe := util.NewStdStreamJoint(reader, writer, exitOnClose)
	lis := util.NewStdinListener()
	done := make(chan error)

	go func() {
		s := grpc.NewServer()

		remote.RegisterTunnelServer(s, NewServer())
		reflection.Register(s)

		done <- s.Serve(lis)
	}()

	lis.Ready(pipe)
	return <-done
}

func NewServer() *tunnelServer {
	return &tunnelServer{}
}

func SendData(stream remote.Tunnel_InitTunnelServer, sessions <-chan *Session, closeChan chan<- bool) {
	for {
		select {
		case <-stream.Context().Done():
			return
		case session := <-sessions:
			// read the bytes from the buffer
			// but allow it to keep growing while we send the response
			session.Lock()
			bys := session.Buf.Len()
			bytes := make([]byte, bys)
			_, _ = session.Buf.Read(bytes)
			resp := &remote.SocketDataResponse{
				HasErr:      false,
				LogMessage:  nil,
				Data:        bytes,
				RequestId:   session.Id.String(),
				ShouldClose: !session.Open,
			}
			session.Unlock()

			logDebugf("sending %d bytes to client", len(bytes))
			err := stream.Send(resp)
			if err != nil {
				logErrorf("failed sending message to tunnel stream")
				closeChan <- true
				return
			}
			logDebugf("sent %d bytes to client", len(bytes))
		}
	}
}

func ReceiveData(stream remote.Tunnel_InitTunnelServer, closeChan chan<- bool) {
	for {
		select {
		case <-stream.Context().Done():
			return
		default:
			message, err := stream.Recv()
			if err != nil {
				logErrorf("failed receiving message from stream, exiting: %v", err)
				closeChan <- true
				continue
			}

			reqId, err := uuid.Parse(message.GetRequestId())
			if err != nil {
				logErrorf(" %s; failed to parse requestId, %v", message.GetRequestId(), err)
				continue
			}

			session, ok := GetSession(reqId)
			if ok != true && !message.ShouldClose {
				logErrorf("%s; session not found in openRequests", reqId)
				continue
			}

			data := message.GetData()
			br := len(data)

			logDebugf("received %d bytes from client", len(data))

			// send data if we received any
			if br > 0 && session.Open {
				logDebugf("writing %d bytes to conn", br)
				_, err := session.Conn.Write(data)
				if err != nil {
					logErrorf("%s; failed writing data to socket", reqId)
					message.ShouldClose = true
				} else {
					logDebugf("wrote %d bytes to conn", br)
				}
			}

			if message.ShouldClose == true {
				logDebugf("closing session")
				session.Close()
				logDebugf("closed session")
			}
		}
	}
}

func readConn(ctx context.Context, session *Session, sessions chan<- *Session) {
	for {
		buff := make([]byte, BufferSize)
		br, err := session.Conn.Read(buff)

		select {
		case <-ctx.Done():
			logDebugf("closing connection")
			session.Close()
			return
		default:
			session.Lock()
			if err != nil {
				if err != io.EOF {
					logErrorf("failed to read from conn: %v", err)
				}

				// setting Open to false triggers SendData() to
				// send ShouldClose
				session.Open = false
			}

			// write the data to the session buffer, if we have data
			if br > 0 {
				session.Buf.Write(buff[0:br])
			}
			session.Unlock()

			sessions <- session
			if session.Open == false {
				return
			}
		}
	}
}

func (t *tunnelServer) InitTunnel(stream remote.Tunnel_InitTunnelServer) error {
	request, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed receiving initial connection from tunnel")
	}
	port := request.GetPort()
	if port == 0 {
		err := stream.Send(&remote.SocketDataResponse{
			HasErr: true,
			LogMessage: &remote.LogMessage{
				LogLevel: remote.LogLevel_ERROR,
				Message:  "missing port",
			},
		})
		if err != nil {
			return err
		}
		return errors.New("missing port")
	}

	ln, err := net.Listen(strings.ToLower(request.GetScheme().String()), fmt.Sprintf(":%d", port))
	if err != nil {
		_ = stream.Send(&remote.SocketDataResponse{
			HasErr: true,
			LogMessage: &remote.LogMessage{
				LogLevel: remote.LogLevel_ERROR,
				Message:  fmt.Sprintf("failed opening listener type %s on port %d: %v", request.GetScheme(), request.GetPort(), err),
			},
		})
		return fmt.Errorf("failed listening on port %d: %v", port, err)
	}

	sessions := make(chan *Session)
	closeChan := make(chan bool, 1)
	go func(close <-chan bool) {
		<-close
		_ = ln.Close()
	}(closeChan)

	go ReceiveData(stream, closeChan)
	go SendData(stream, sessions, closeChan)

	for {
		connection, err := ln.Accept()
		if err != nil {
			return err
		}
		logDebugf("accepted new connection on ::%d", port)

		// socket -> stream
		session, err := NewSession(connection)
		if err != nil {
			logErrorf("create new session: %v", err)
			continue
		}

		go readConn(stream.Context(), session, sessions)
	}
}
