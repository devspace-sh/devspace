package server

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/ptr"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsStream struct {
	WebSocket *websocket.Conn

	writeMutex sync.Mutex
	readMutex  sync.Mutex
}

func (ws *wsStream) Write(p []byte) (int, error) {
	ws.writeMutex.Lock()
	defer ws.writeMutex.Unlock()

	err := ws.WebSocket.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (ws *wsStream) Read(p []byte) (int, error) {
	ws.readMutex.Lock()
	defer ws.readMutex.Unlock()

	ws.WebSocket.SetReadLimit(int64(len(p)))
	_, message, err := ws.WebSocket.ReadMessage()
	if err != nil {
		return 0, err
	}

	copy(p, message)
	return len(message), nil
}

func (h *handler) logs(w http.ResponseWriter, r *http.Request) {
	// Kube Context
	kubeContext := h.ctx.KubeClient().CurrentContext()
	contextParam, ok := r.URL.Query()["context"]
	if ok && len(contextParam) == 1 && contextParam[0] != "" {
		kubeContext = contextParam[0]
	}

	// Namespace
	kubeNamespace := h.ctx.KubeClient().Namespace()
	namespace, ok := r.URL.Query()["namespace"]
	if ok && len(namespace) == 1 && namespace[0] != "" {
		kubeNamespace = namespace[0]
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(kubeContext, kubeNamespace, false, kubeconfig.NewLoader())
	if err != nil {
		h.ctx.Log().Errorf("Error in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name, ok := r.URL.Query()["name"]
	if !ok || len(name) != 1 {
		http.Error(w, "name is missing", http.StatusBadRequest)
		return
	}
	container, ok := r.URL.Query()["container"]
	if !ok || len(container) != 1 {
		http.Error(w, "container is missing", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.ctx.Log().Errorf("Error upgrading connection in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer ws.Close()

	// Open logs connection
	reader, err := client.Logs(context.TODO(), namespace[0], name[0], container[0], false, ptr.Int64(100), true)
	if err != nil {
		h.ctx.Log().Errorf("Error in %s: %v", r.URL.String(), err)
		websocketError(ws, err)
		return
	}

	defer reader.Close()

	// Stream logs
	stream := &wsStream{WebSocket: ws}
	_, err = io.Copy(stream, reader)
	if err != nil {
		h.ctx.Log().Errorf("Error in %s pipeReader: %v", r.URL.String(), err)
		websocketError(ws, err)
		return
	}

	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 5))
	_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
