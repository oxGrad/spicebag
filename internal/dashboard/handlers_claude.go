package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

const sessionBufCap = 128 * 1024

type claudeSession struct {
	id   string
	cmd  *exec.Cmd
	ptmx *os.File

	mu     sync.Mutex
	buf    []byte
	exited bool
	conn   *websocket.Conn // current subscriber; nil when panel is closed
}

// getOrCreateSession returns an existing live session or starts a new one.
func (s *Server) getOrCreateSession(id string) (*claudeSession, error) {
	s.sessmu.Lock()
	defer s.sessmu.Unlock()

	if sess, ok := s.sessions[id]; ok {
		return sess, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(claudePath)
	cmd.Dir = filepath.Join(home, ".config", "spicebag")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	sess := &claudeSession{id: id, cmd: cmd, ptmx: ptmx}
	s.sessions[id] = sess

	go s.runSession(sess)

	return sess, nil
}

// runSession reads PTY output, buffers it, and forwards to any active subscriber.
func (s *Server) runSession(sess *claudeSession) {
	tmp := make([]byte, 4096)
	for {
		n, rdErr := sess.ptmx.Read(tmp)
		if n > 0 {
			data := make([]byte, n)
			copy(data, tmp[:n])

			sess.mu.Lock()
			sess.buf = append(sess.buf, data...)
			if len(sess.buf) > sessionBufCap {
				sess.buf = sess.buf[len(sess.buf)-sessionBufCap:]
			}
			conn := sess.conn
			sess.mu.Unlock()

			if conn != nil {
				conn.WriteMessage(websocket.BinaryMessage, data)
			}
		}
		if rdErr != nil {
			sess.mu.Lock()
			sess.exited = true
			conn := sess.conn
			sess.mu.Unlock()

			if conn != nil {
				sendWSControl(conn, "exit", "")
			}
			// Keep session in map briefly so the client can read the exit status,
			// then clean up.
			time.AfterFunc(10*time.Minute, func() {
				s.sessmu.Lock()
				if s.sessions[sess.id] == sess {
					delete(s.sessions, sess.id)
				}
				s.sessmu.Unlock()
			})
			return
		}
	}
}

func (s *Server) killSession(id string) {
	s.sessmu.Lock()
	sess, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.sessmu.Unlock()
	if ok {
		sess.ptmx.Close()
		sess.cmd.Process.Kill()
		sess.cmd.Wait()
	}
}

func (s *Server) handleWSClaude(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		sendWSControl(conn, "error", "session ID required")
		return
	}

	sess, err := s.getOrCreateSession(sessionID)
	if err != nil {
		sendWSControl(conn, "error", err.Error())
		return
	}

	// Register subscriber, replay buffer.
	sess.mu.Lock()
	sess.conn = conn
	buf := make([]byte, len(sess.buf))
	copy(buf, sess.buf)
	exited := sess.exited
	sess.mu.Unlock()

	if len(buf) > 0 {
		conn.WriteMessage(websocket.BinaryMessage, buf)
	}
	if exited {
		sendWSControl(conn, "exit", "")
	}

	// Handle incoming messages.
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		switch msgType {
		case websocket.BinaryMessage:
			sess.ptmx.Write(data)
		case websocket.TextMessage:
			var msg struct {
				Type string `json:"type"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal(data, &msg) == nil {
				switch msg.Type {
				case "resize":
					if msg.Cols > 0 && msg.Rows > 0 {
						pty.Setsize(sess.ptmx, &pty.Winsize{Cols: msg.Cols, Rows: msg.Rows})
					}
				case "kill":
					s.killSession(sessionID)
				}
			}
		}
	}

	// WebSocket closed — unsubscribe but keep session alive.
	sess.mu.Lock()
	if sess.conn == conn {
		sess.conn = nil
	}
	sess.mu.Unlock()
}

func sendWSControl(conn *websocket.Conn, kind, detail string) {
	msg := struct {
		Type   string `json:"type"`
		Detail string `json:"detail,omitempty"`
	}{Type: kind, Detail: detail}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}
