package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"devhub/internal/terminal"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type terminalControlMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// HandleTerminalWS upgrades to WebSocket and bridges to a PTY session.
func HandleTerminalWS(manager *terminal.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		sess, ok := manager.Get(id)
		if !ok {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("terminal ws upgrade error: %v", err)
			return
		}

		var closeOnce sync.Once
		cleanup := func() {
			closeOnce.Do(func() {
				conn.Close()
				// Don't destroy the PTY session here -- the browser may
				// reconnect after a split/remount.  Sessions are destroyed
				// only via REST DELETE or server shutdown.
			})
		}
		defer cleanup()

		// PTY -> WebSocket (binary frames)
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := sess.Pty.Read(buf)
				if n > 0 {
					if wErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); wErr != nil {
						cleanup()
						return
					}
				}
				if err != nil {
					exitCode := 0
					if err != io.EOF {
						exitCode = 1
					}
					exitMsg, _ := json.Marshal(map[string]interface{}{
						"type": "exit",
						"code": exitCode,
					})
					conn.WriteMessage(websocket.TextMessage, exitMsg)
					cleanup()
					return
				}
			}
		}()

		// WebSocket -> PTY
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				cleanup()
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				if _, err := sess.Pty.Write(data); err != nil {
					cleanup()
					return
				}
			case websocket.TextMessage:
				var msg terminalControlMsg
				if err := json.Unmarshal(data, &msg); err != nil {
					continue
				}
				if msg.Type == "resize" && msg.Cols > 0 && msg.Rows > 0 {
					sess.Resize(msg.Cols, msg.Rows)
				}
			}
		}
	}
}
