package tunnel

import (
	"fmt"
	"github.com/devspace-cloud/devspace/helper/remote"
	"github.com/devspace-cloud/devspace/helper/util"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
	"net"
	"os"
	"strings"
)

type tunnelServer struct{}

const (
	bufferSize = 1024 * 32
)

func logErrorf(message string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, message, args...)
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

func SendData(stream *remote.Tunnel_InitTunnelServer, sessions <-chan *Session, closeChan chan<- bool) {
	for {
		session := <-sessions
		session.Lock.Lock()
		resp := &remote.SocketDataResponse{
			HasErr:      false,
			LogMessage:  nil,
			RequestId:   session.Id.String(),
			Data:        session.Buf.Bytes(),
			ShouldClose: !session.Open,
		}
		session.Buf.Reset()
		session.Lock.Unlock()
		st := *stream
		err := st.Send(resp)
		if err != nil {
			logErrorf("failed sending message to tunnel stream, exitin ; %v", err)
			closeChan <- true
			return
		}
	}
}

func ReceiveData(stream *remote.Tunnel_InitTunnelServer, closeChan chan<- bool) {
	st := *stream
	for {
		message, err := st.Recv()
		if err != nil {
			logErrorf("failed receiving message from stream, exiting: %v", err)
			closeChan <- true
			return
		}
		reqId, err := uuid.Parse(message.GetRequestId())
		if err != nil {
			logErrorf(" %s; failed to parse requestId, %v", message.GetRequestId(), err)
		} else {
			session, ok := GetSession(reqId)
			if ok != true {
				logErrorf("%s; session not found in openRequests", reqId)
			} else {
				data := message.GetData()
				if len(data) > 0 {
					conn := session.Conn
					_, err := conn.Write(data)
					if err != nil {
						logErrorf("%s; failed writing data to socket", reqId)
					}
				}
				if message.ShouldClose == true {
					ok, _ := CloseSession(reqId)
					if ok != true {
						logErrorf("%s; failed closing session", reqId)
					}
				}
			}
		}
	}
}

func readConn(session *Session, sessions chan<- *Session) {
	sessions <- session
	// We want to inform the client that we accepted a connection - some weird ass protocols wait for data from the server when connecting
	// Read from socket in a loop and push messages to the sessions channel
	// If the socket is closed, signal the channel to close connection
	buff := make([]byte, bufferSize)
	for {
		br, err := session.Conn.Read(buff)
		session.Lock.Lock()
		logErrorf("read %d bytes from socket, err: %v", br, err)
		if err != nil {
			session.Open = false
		}
		if br > 0 {
			session.Buf.Write(buff[:br])
			if br == len(buff) {
				newSize := len(buff) * 2
				buff = make([]byte, newSize)
			}
		}
		session.Lock.Unlock()
		if !session.Open {
			_, _ = CloseSession(session.Id)
			return
		}
		sessions <- session
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

	go ReceiveData(&stream, closeChan)
	go SendData(&stream, sessions, closeChan)

	for {
		connection, err := ln.Accept()
		if err != nil {
			return err
		}
		// socket -> stream
		session := NewSession(connection)
		go readConn(session, sessions)
	}
}
