package server

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func pipeReader(ws *websocket.Conn, r io.Reader) error {
	s := bufio.NewScanner(r)
	for s.Scan() {
		// ws.SetWriteDeadline(time.Now().Add(writeWait))
		if err := ws.WriteMessage(websocket.BinaryMessage, s.Bytes()); err != nil {
			ws.Close()
			break
		}
	}
	if s.Err() != nil {
		return s.Err()
	}

	ws.SetWriteDeadline(time.Now().Add(time.Second * 5))
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	ws.Close()
	return nil
}

func (h *handler) logs(w http.ResponseWriter, r *http.Request) {
	name, ok := r.URL.Query()["name"]
	if !ok || len(name) != 1 {
		http.Error(w, "name is missing", http.StatusBadRequest)
		return
	}
	namespace, ok := r.URL.Query()["namespace"]
	if !ok || len(namespace) != 1 {
		http.Error(w, "namespace is missing", http.StatusBadRequest)
		return
	}
	container, ok := r.URL.Query()["container"]
	if !ok || len(container) != 1 {
		http.Error(w, "container is missing", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorf("Error upgrading connection: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer ws.Close()

	// Open logs connection
	reader, err := h.client.Logs(context.Background(), namespace[0], name[0], container[0], false, ptr.Int64(100), true)
	if err != nil {
		h.log.Errorf("Error in /api/logs logs: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer reader.Close()

	// Stream logs
	err = pipeReader(ws, reader)
	if err != nil {
		h.log.Errorf("Error in /api/logs pipeReader: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
