package server

import (
	"io"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
)

func websocketError(ws *websocket.Conn, err error) {
	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 2))
	_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
}

func (h *handler) command(w http.ResponseWriter, r *http.Request) {
	name, ok := r.URL.Query()["name"]
	if !ok || len(name) != 1 {
		http.Error(w, "name is missing", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.ctx.Log().Errorf("Error upgrading connection: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ws.Close()

	// Open logs connection
	stream := &wsStream{WebSocket: ws}
	cmd := exec.Command("devspace", "--namespace", h.ctx.KubeClient().Namespace(), "--kube-context", h.ctx.KubeClient().CurrentContext(), "run", name[0])
	done := make(chan bool)
	defer close(done)

	stdinWriter, err := cmd.StdinPipe()
	if err != nil {
		return
	}

	defer stdinWriter.Close()

	cmd.Stdout = stream
	cmd.Stderr = stream

	go func(done chan bool) {
		_, _ = io.Copy(stdinWriter, stream)

		select {
		case <-done:
		case <-time.After(time.Second):
			proc := cmd.Process
			if proc != nil {
				_ = proc.Kill()
			}
		}
	}(done)

	err = cmd.Run()
	if err != nil {
		h.ctx.Log().Errorf("Error in %s: %v", r.URL.String(), err)
		websocketError(ws, err)
		return
	}

	_ = ws.SetWriteDeadline(time.Now().Add(time.Second * 5))
	_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
