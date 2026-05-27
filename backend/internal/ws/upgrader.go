package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/bongcloud/chess/internal/config"
)

// Upgrader is a configured gorilla/websocket upgrader that is safe for
// concurrent use. Inject it into handlers that need to upgrade HTTP connections.
type Upgrader struct {
	upgrader websocket.Upgrader
	cfg      config.WebSocketConfig
	log      *zap.Logger
}

// NewUpgrader builds an Upgrader from application config.
func NewUpgrader(cfg config.WebSocketConfig, log *zap.Logger) *Upgrader {
	return &Upgrader{
		cfg: cfg,
		log: log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:   cfg.ReadBufferSize,
			WriteBufferSize:  cfg.WriteBufferSize,
			HandshakeTimeout: cfg.HandshakeTimeout,
			CheckOrigin: func(r *http.Request) bool { return true },
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				log.Warn("websocket upgrade failed",
					zap.Int("status", status),
					zap.Error(reason),
				)
			},
		},
	}
}

// Upgrade performs the HTTP -> WebSocket handshake and returns a *Conn wrapper.
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, header http.Header) (*Conn, error) {
	raw, err := u.upgrader.Upgrade(w, r, header)
	if err != nil {
		return nil, err
	}
	return newConn(raw, u.cfg, u.log), nil
}

// Conn wraps *websocket.Conn with deadline management helpers and structured
// logging. Business logic (game hub, matchmaking) builds on top of this layer.
type Conn struct {
	raw  *websocket.Conn
	cfg  config.WebSocketConfig
	log  *zap.Logger
	send chan []byte
}

func newConn(raw *websocket.Conn, cfg config.WebSocketConfig, log *zap.Logger) *Conn {
	c := &Conn{
		raw:  raw,
		cfg:  cfg,
		log:  log,
		send: make(chan []byte, 256),
	}
	// Set the initial read deadline so idle connections are cleaned up
	_ = raw.SetReadDeadline(time.Now().Add(cfg.PongWait))
	raw.SetPongHandler(func(string) error {
		return raw.SetReadDeadline(time.Now().Add(cfg.PongWait))
	})
	return c
}

// ReadMessage blocks until a message arrives or an error occurs.
func (c *Conn) ReadMessage() (messageType int, p []byte, err error) {
	return c.raw.ReadMessage()
}

// WriteMessage writes a message with a short write deadline.
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	_ = c.raw.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.raw.WriteMessage(messageType, data)
}

// WritePing sends a WebSocket ping control frame.
func (c *Conn) WritePing() error {
	_ = c.raw.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.raw.WriteMessage(websocket.PingMessage, nil)
}

// Close terminates the underlying connection.
func (c *Conn) Close() error {
	return c.raw.Close()
}

// RemoteAddr returns the remote network address of the connection.
func (c *Conn) RemoteAddr() string {
	return c.raw.RemoteAddr().String()
}