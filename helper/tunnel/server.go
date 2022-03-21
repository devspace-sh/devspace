package tunnel

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/loft-sh/devspace/helper/remote"
	"github.com/loft-sh/devspace/helper/util"
	"github.com/loft-sh/devspace/helper/util/pingtimeout"
	"github.com/loft-sh/devspace/helper/util/stderrlog"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
	"net"
	"os"
	"strings"
)

type tunnelServer struct {
	remote.UnimplementedTunnelServer

	// ping is used to determine if we still have an alive connection
	ping *pingtimeout.PingTimeout
}

func StartTunnelServer(reader io.Reader, writer io.Writer, exitOnClose, ping bool) error {
	pipe := util.NewStdStreamJoint(reader, writer, exitOnClose)
	lis := util.NewStdinListener()
	done := make(chan error)

	go func() {
		s := grpc.NewServer()

		tunnel := &tunnelServer{
			ping: &pingtimeout.PingTimeout{},
		}

		if ping {
			doneChan := make(chan struct{})
			defer close(doneChan)
			tunnel.ping.Start(doneChan)
		}

		remote.RegisterTunnelServer(s, tunnel)
		reflection.Register(s)

		done <- s.Serve(lis)
	}()

	lis.Ready(pipe)
	return <-done
}

func SendData(stream remote.Tunnel_InitTunnelServer, sessions <-chan *Session, closeChan chan struct{}) {
	for {
		select {
		case <-closeChan:
			return
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
				RequestId:   session.ID.String(),
				ShouldClose: !session.Open,
			}
			session.Unlock()

			stderrlog.Debugf("sending %d bytes to client", len(bytes))
			err := stream.Send(resp)
			if err != nil {
				stderrlog.Errorf("failed sending message to tunnel stream: %v", err)
				close(closeChan)
				return
			}
			stderrlog.Debugf("sent %d bytes to client", len(bytes))
		}
	}
}

func ReceiveData(stream remote.Tunnel_InitTunnelServer, closeChan chan struct{}) {
	for {
		select {
		case <-closeChan:
			return
		case <-stream.Context().Done():
			return
		default:
			message, err := stream.Recv()
			if err != nil {
				stderrlog.Errorf("failed receiving message from stream, exiting: %v", err)
				close(closeChan)
				continue
			}

			reqID, err := uuid.Parse(message.GetRequestId())
			if err != nil {
				stderrlog.Errorf(" %s; failed to parse requestId, %v", message.GetRequestId(), err)
				continue
			}

			session, ok := GetSession(reqID)
			if !ok && !message.ShouldClose {
				stderrlog.Errorf("%s; session not found in openRequests", reqID)
				continue
			}

			data := message.GetData()
			br := len(data)

			stderrlog.Debugf("received %d bytes from client", len(data))

			// send data if we received any
			if br > 0 && session.Open {
				stderrlog.Debugf("writing %d bytes to conn", br)
				_, err := session.Conn.Write(data)
				if err != nil {
					stderrlog.Errorf("%s; failed writing data to socket", reqID)
					message.ShouldClose = true
				} else {
					stderrlog.Debugf("wrote %d bytes to conn", br)
				}
			}

			if message.ShouldClose {
				stderrlog.Debugf("closing session")
				session.Close()
				stderrlog.Debugf("closed session")
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
			stderrlog.Debugf("closing connection")
			session.Close()
			return
		default:
			session.Lock()
			if err != nil {
				if err != io.EOF {
					stderrlog.Errorf("failed to read from conn: %v", err)
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
			if !session.Open {
				return
			}
		}
	}
}

// Ping returns empty
func (t *tunnelServer) Ping(context.Context, *remote.Empty) (*remote.Empty, error) {
	if t.ping != nil {
		t.ping.Ping()
	}

	return &remote.Empty{}, nil
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
	closeChan := make(chan struct{})
	go func(close chan struct{}) {
		<-close
		_ = ln.Close()
		os.Exit(1)
	}(closeChan)

	go ReceiveData(stream, closeChan)
	go SendData(stream, sessions, closeChan)

	for {
		connection, err := ln.Accept()
		if err != nil {
			return err
		}
		stderrlog.Debugf("accepted new connection on ::%d", port)

		// socket -> stream
		session, err := NewSession(connection)
		if err != nil {
			stderrlog.Errorf("create new session: %v", err)
			continue
		}

		go readConn(stream.Context(), session, sessions)
	}
}
