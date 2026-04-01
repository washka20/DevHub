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

		// Stop any previous reader goroutine before starting a new one.
		// This prevents two goroutines reading the same PTY concurrently
		// (happens on split/remount when the browser reconnects).
		sess.StopReader()

		var closeOnce sync.Once
		cleanup := func() {
			closeOnce.Do(func() {
				conn.Close()
			})
		}
		defer cleanup()

		// Scrollback replay: send tail of log file before starting live stream.
		logPath := filepath.Join(os.TempDir(), "devhub-terminal-logs", id+".log")
		if f, err := os.Open(logPath); err == nil {
			defer f.Close()
			const maxReplay = 64 * 1024
			if stat, err := f.Stat(); err == nil && stat.Size() > 0 {
				offset := int64(0)
				if stat.Size() > maxReplay {
					offset = stat.Size() - maxReplay
					// Scan forward to next newline to avoid mid-sequence cut
					if _, seekErr := f.Seek(offset, io.SeekStart); seekErr != nil {
						log.Printf("terminal scrollback seek error: %v", seekErr)
						offset = 0
					} else {
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
				} else {
					buf := make([]byte, 4096)
					for {
						n, err := f.Read(buf)
						if n > 0 {
							if wErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); wErr != nil {
								cleanup()
								return
							}
						}
						if err != nil {
							break
						}
					}
				}
			}
		}

		// PTY -> WebSocket (binary frames)
		stopCh := sess.StartReader()
		go func() {
			defer sess.ReaderDone()
			buf := make([]byte, 4096)
			for {
				select {
				case <-stopCh:
					return
				default:
				}

				n, err := sess.Pty.Read(buf)
				if n > 0 {
					data := buf[:n]
					if wErr := conn.WriteMessage(websocket.BinaryMessage, data); wErr != nil {
						cleanup()
						return
					}
					// Persist output to log file (best-effort, ignore errors)
					if sess.LogFile != nil {
						sess.LogFile.Write(data)
					}
				}
				if err != nil {
					// PTY closed (shell exited)
					exitCode := 0
					if err != io.EOF {
						exitCode = 1
					}
					exitMsg, _ := json.Marshal(map[string]interface{}{
						"type": "exit",
						"code": exitCode,
					})
					conn.WriteMessage(websocket.TextMessage, exitMsg)
					// Shell exited -- destroy the session so it doesn't linger
					manager.Destroy(id)
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
