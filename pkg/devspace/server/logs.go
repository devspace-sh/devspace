package server

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func pipeReader(ws *websocket.Conn, r io.Reader) error {
	defer log.Info("Done bois")

	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if err := ws.WriteMessage(websocket.BinaryMessage, b[:n]); err != nil {
			ws.Close()
			return err
		}
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
