package tunnel

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Session struct {
	Id   uuid.UUID
	Conn net.Conn
	Buf  bytes.Buffer
	Open bool
	Lock sync.RWMutex
}

type RedirectRequest struct {
	Source int32
	Target int32
}

func NewSession(conn net.Conn) *Session {
	r := &Session{
		Id:   uuid.New(),
		Conn: conn,
		Buf:  bytes.Buffer{},
		Open: true,
	}
	ok, err := AddSession(r)
	if ok != true {
		logErrorf("%s; failed registering request: %v", r.Id.String(), err)
	}
	return r
}

func NewSessionFromStream(id uuid.UUID, conn net.Conn) *Session {
	r := &Session{
		Id:   id,
		Conn: conn,
		Buf:  bytes.Buffer{},
		Open: true,
	}
	ok, err := AddSession(r)
	if ok != true {
		logErrorf("%s; failed registering request: %v", r.Id.String(), err)
	}
	return r
}

func AddSession(r *Session) (bool, error) {
	if _, ok := GetSession(r.Id); ok != false {
		return false, errors.New(fmt.Sprintf("Session %s already exists", r.Id.String()))
	}
	openSessions.Store(r.Id, r)
	return true, nil
}

func GetSession(id uuid.UUID) (*Session, bool) {
	request, ok := openSessions.Load(id)
	if ok {
		return request.(*Session), ok
	}
	return nil, ok
}

var openSessions = sync.Map{}

func CloseSession(id uuid.UUID) (bool, error) {
	session, ok := GetSession(id)
	if ok == false {
		return true, nil
	}
	session.Lock.Lock()
	conn := session.Conn
	err := conn.Close()
	session.Lock.Unlock()
	openSessions.Delete(id)
	return true, err
}

func ParsePorts(s string) (*RedirectRequest, error) {
	raw := strings.Split(s, ":")
	if len(raw) == 0 {
		return nil, errors.New(fmt.Sprintf("failed parsing redirect request: %s", s))
	}
	if len(raw) == 1 {
		p, err := strconv.ParseInt(raw[0], 10, 32)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to parse port %s, %v", raw[0], err))
		}
		return &RedirectRequest{
			Source: int32(p),
			Target: int32(p),
		}, nil
	}
	if len(raw) == 2 {
		s, err := strconv.ParseInt(raw[0], 10, 32)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to parse port %s, %v", raw[0], err))
		}
		t, err := strconv.ParseInt(raw[1], 10, 32)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("failed to parse port %s, %v", raw[1], err))
		}
		return &RedirectRequest{
			Source: int32(s),
			Target: int32(t),
		}, nil
	}
	return nil, errors.New(fmt.Sprintf("Error, bad tunnel format: %s", s))
}
