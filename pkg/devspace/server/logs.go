package server

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/gorilla/websocket"
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

func (h *handler) logsMultiple(w http.ResponseWriter, r *http.Request) {
	// Kube Context
	kubeContext := h.defaultContext
	context, ok := r.URL.Query()["context"]
	if ok && len(context) == 1 && context[0] != "" {
		kubeContext = context[0]
	}

	// Namespace
	kubeNamespace := h.defaultNamespace
	namespace, ok := r.URL.Query()["namespace"]
	if ok && len(namespace) == 1 && namespace[0] != "" {
		kubeNamespace = namespace[0]
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(kubeContext, kubeNamespace, false, kubeconfig.NewLoader())
	if err != nil {
		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	imageSelector, ok := r.URL.Query()["imageSelector"]
	if !ok || len(imageSelector) == 0 {
		http.Error(w, "imageSelector is missing", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorf("Error upgrading connection in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer ws.Close()

	writer := &wsStream{WebSocket: ws}
	err = client.LogMultipleTimeout(imageSelector, make(chan error), ptr.Int64(100), writer, 0, log.Discard)
	if err != nil {
		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		websocketError(ws, err)
		return
	}

	ws.SetWriteDeadline(time.Now().Add(time.Second * 5))
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (h *handler) logs(w http.ResponseWriter, r *http.Request) {
	// Kube Context
	kubeContext := h.defaultContext
	contextParam, ok := r.URL.Query()["context"]
	if ok && len(contextParam) == 1 && contextParam[0] != "" {
		kubeContext = contextParam[0]
	}

	// Namespace
	kubeNamespace := h.defaultNamespace
	namespace, ok := r.URL.Query()["namespace"]
	if ok && len(namespace) == 1 && namespace[0] != "" {
		kubeNamespace = namespace[0]
	}

	// Create kubectl client
	client, err := kubectl.NewClientFromContext(kubeContext, kubeNamespace, false, kubeconfig.NewLoader())
	if err != nil {
		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
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
		h.log.Errorf("Error upgrading connection in %s: %v", r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer ws.Close()

	// Open logs connection
	reader, err := client.Logs(context.Background(), namespace[0], name[0], container[0], false, ptr.Int64(100), true)
	if err != nil {
		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		websocketError(ws, err)
		return
	}

	defer reader.Close()

	// Stream logs
	stream := &wsStream{WebSocket: ws}
	_, err = io.Copy(stream, reader)
	if err != nil {
		h.log.Errorf("Error in %s pipeReader: %v", r.URL.String(), err)
		websocketError(ws, err)
		return
	}

	ws.SetWriteDeadline(time.Now().Add(time.Second * 5))
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
