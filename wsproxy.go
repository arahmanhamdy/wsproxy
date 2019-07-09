package wsproxy

import (
	"net"
	"net/http"
	"time"
	"github.com/gorilla/websocket"
	"github.com/caddyserver/caddy/caddyhttp/httpserver"
	"bytes"
	"bufio"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024 * 10 // 10 MB default.
)

type (
	// WebSocket is a type that holds configuration for the
	// websocket middleware generally, like a list of all the
	// websocket endpoints.
	WebSocket struct {
		// Next is the next HTTP handler in the chain for when the path doesn't match
		Next    httpserver.Handler

		// Sockets holds all the web socket endpoint configurations
		Sockets []Config
	}

	// Config holds the configuration for a single websocket
	// endpoint which may serve multiple websocket connections.
	Config struct {
		Path          string
		TCPSocketAddr string
	}
)

// ServeHTTP converts the HTTP request to a WebSocket connection and serves it up.
func (ws WebSocket) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	for _, sockconfig := range ws.Sockets {
		if httpserver.Path(r.URL.Path).Matches(sockconfig.Path) {
			return serveWS(w, r, &sockconfig)
		}
	}

	// Didn't match a websocket path, so pass-through
	return ws.Next.ServeHTTP(w, r)
}

// serveWS is used for setting and upgrading the HTTP connection to a websocket connection.
// It also spawns the child process that is associated with matched HTTP path/url.
func serveWS(w http.ResponseWriter, r *http.Request, config *Config) (int, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// the connection has been "handled" -- WriteHeader was called with Upgrade,
		// so don't return an error status code; just return an error
		return 0, err
	}
	defer conn.Close()

	// Open The TCP Socket Here
	tcpSocket, err := net.Dial("tcp", config.TCPSocketAddr)

	defer tcpSocket.Close()

	if err != nil {
		return 0, err
	}

	done := make(chan struct{})
	go tcp2ws(conn, tcpSocket, done)
	ws2tcp(conn, tcpSocket)

	return 0, nil
}

// ws2tcp handles reading data from the websocket connection and writing
// it to tcp socket.
func ws2tcp(conn *websocket.Conn, tcpSocket net.Conn) {
	// Setup our connection's websocket ping/pong handlers from our const values.
	defer conn.Close()
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait)); return nil
	})
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if _, err := tcpSocket.Write(message); err != nil {
			break
		}
	}
}

// tcp2ws handles reading data from tcp socket and writing
// it to websocket connection.
func tcp2ws(conn *websocket.Conn, tcpSocket net.Conn, done chan struct{}) {
	go pinger(conn, done)
	defer func() {
		conn.Close()
		close(done) // make sure to close the pinger when we are done.
	}()

	/*s := bufio.NewScanner(tcpSocket)
	for s.Scan() {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, bytes.TrimSpace(s.Bytes())); err != nil {
			break
		}
	}
	if s.Err() != nil {
		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, s.Err().Error()), time.Time{})
	}*/
	reader := bufio.NewReader(tcpSocket)
	data := make([]byte, 4096)
	for {
		_, err := reader.Read(data)
		if err != nil {
			conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, err.Error()), time.Time{})
		}
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, bytes.Trim(data, "\x00")); err != nil {
			break
		}
	}
}

// pinger simulates the websocket to keep it alive with ping messages.
func pinger(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		// blocking loop with select to wait for stimulation.
		select {
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, err.Error()), time.Time{})
				return
			}
		case <-done:
			return // clean up this routine.
		}
	}
}
