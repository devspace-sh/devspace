package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	BufferSize = 1024 * 10
)

var openSessions = sync.Map{}

type Session struct {
	ID         uuid.UUID
	Conn       net.Conn
	Buf        bytes.Buffer
	Context    context.Context
	cancelFunc context.CancelFunc
	Open       bool
	sync.Mutex
}

func (s *Session) Close() {
	s.cancelFunc()
	if s.Conn != nil {
		_ = s.Conn.Close()
		s.Open = false
	}
	go func() {
		<-time.After(5 * time.Second)
		openSessions.Delete(s.ID)
	}()
}

type RedirectRequest struct {
	Source int32
	Target int32
}

func NewSession(conn net.Conn) (*Session, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Session{
		ID:         uuid.New(),
		Conn:       conn,
		Context:    ctx,
		cancelFunc: cancel,
		Buf:        bytes.Buffer{},
		Open:       true,
	}
	err := addSession(r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func NewSessionFromStream(id uuid.UUID, conn net.Conn) (*Session, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Session{
		ID:         id,
		Conn:       conn,
		Context:    ctx,
		cancelFunc: cancel,
		Buf:        bytes.Buffer{},
		Open:       true,
	}
	err := addSession(r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func addSession(r *Session) error {
	if _, ok := GetSession(r.ID); ok {
		return fmt.Errorf("session %s already exists", r.ID.String())
	}
	openSessions.Store(r.ID, r)
	return nil
}

func GetSession(id uuid.UUID) (*Session, bool) {
	request, ok := openSessions.Load(id)
	if ok {
		return request.(*Session), ok
	}
	return nil, ok
}
