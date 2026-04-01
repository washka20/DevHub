package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
// The session has a persistent pump goroutine that reads PTY output; this
// handler only needs to attach/detach the WS connection and relay input.
func HandleTerminalWS(manager *terminal.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		if len(id) > 64 {
			http.Error(w, "invalid session id", http.StatusBadRequest)
			return
		}
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
		log.Printf("terminal ws: connected to session %s", id)

		var closeOnce sync.Once
		cleanup := func() {
			closeOnce.Do(func() {
				conn.Close()
			})
		}
		defer cleanup()

		// Detach any previous WS output (previous browser tab / reconnect)
		sess.DetachOutput()

		// Check if shell already exited (e.g. user reconnects to dead session)
		select {
		case <-sess.ExitCh():
			log.Printf("terminal ws: session %s already exited, replaying + sending exit", id)
			replayScrollback(conn, id)
			exitMsg, _ := json.Marshal(map[string]interface{}{
				"type": "exit",
				"code": sess.ExitCode(),
			})
			conn.WriteMessage(websocket.TextMessage, exitMsg)
			manager.Destroy(id)
			return
		default:
		}

		// Scrollback replay: send tail of log file before live stream
		replayScrollback(conn, id)

		// Attach live output: pump goroutine will call this for every PTY chunk
		sess.AttachOutput(func(data []byte) {
			if wErr := conn.WriteMessage(websocket.BinaryMessage, data); wErr != nil {
				cleanup()
			}
		})

		// Monitor shell exit in background
		wsHandlerDone := make(chan struct{})
		go func() {
			select {
			case <-sess.ExitCh():
				exitMsg, _ := json.Marshal(map[string]interface{}{
					"type": "exit",
					"code": sess.ExitCode(),
				})
				conn.WriteMessage(websocket.TextMessage, exitMsg)
				manager.Destroy(id)
				cleanup()
			case <-wsHandlerDone:
				// WS handler returned (browser disconnected), stop monitoring
			}
		}()

		// WebSocket -> PTY
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				sess.DetachOutput()
				close(wsHandlerDone)
				return
			}

			switch msgType {
			case websocket.BinaryMessage:
				if _, err := sess.Pty.Write(data); err != nil {
					sess.DetachOutput()
					close(wsHandlerDone)
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

// replayScrollback sends the tail of the session's log file over the WS
// so the user sees previous terminal output on reconnect.
func replayScrollback(conn *websocket.Conn, id string) {
	logPath := filepath.Join(os.TempDir(), "devhub-terminal-logs", id+".log")
	f, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer f.Close()

	const maxReplay = 64 * 1024
	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return
	}

	offset := int64(0)
	if stat.Size() > maxReplay {
		offset = stat.Size() - maxReplay
		if _, seekErr := f.Seek(offset, io.SeekStart); seekErr != nil {
			log.Printf("terminal scrollback seek error: %v", seekErr)
			offset = 0
		} else {
			// Scan forward to next newline to avoid mid-sequence cut
			oneByte := make([]byte, 1)
			for {
				n, err := f.Read(oneByte)
				if n > 0 {
					offset++
					if oneByte[0] == '\n' {
						break
					}
				}
				if err != nil {
					break
				}
			}
		}
	}

	if _, seekErr := f.Seek(offset, io.SeekStart); seekErr != nil {
		log.Printf("terminal scrollback seek error: %v", seekErr)
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if wErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); wErr != nil {
				return
			}
		}
		if err != nil {
			break
		}
	}
}
