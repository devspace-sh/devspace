package server

import (
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
)

func (h *handler) command(w http.ResponseWriter, r *http.Request) {
	name, ok := r.URL.Query()["name"]
	if !ok || len(name) != 1 {
		http.Error(w, "name is missing", http.StatusBadRequest)
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
	cmd := exec.Command("devspace", "run", name[0])

	// The current problem if we pipe the stdin to the command is that the command
	// is not terminating anymore, since it waits forever for stdin to close
	// cmd.Stdin = stream
	cmd.Stdout = stream
	cmd.Stderr = stream

	err = cmd.Run()
	if err != nil {
		ws.SetWriteDeadline(time.Now().Add(time.Second))
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))

		h.log.Errorf("Error in %s: %v", r.URL.String(), err)
		return
	}

	ws.SetWriteDeadline(time.Now().Add(time.Second * 5))
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
