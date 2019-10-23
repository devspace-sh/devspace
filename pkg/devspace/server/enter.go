package server

import (
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gorilla/websocket"
)

func (h *handler) enter(w http.ResponseWriter, r *http.Request) {
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
	stream := &wsStream{WebSocket: ws}
	err = h.client.ExecStream(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name[0],
			Namespace: namespace[0],
		},
	}, container[0], []string{"sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"}, true, stream, stream, stream)
	if err != nil {
		ws.SetWriteDeadline(time.Now().Add(time.Second))
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))

		h.log.Errorf("Error in /api/enter: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ws.SetWriteDeadline(time.Now().Add(time.Second * 5))
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
