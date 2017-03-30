package server

import (
	"errors"
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/synacor/sibyl/game"
	"github.com/synacor/sibyl/name"
)

const (
	readLimit = 2048 // 2KiB

	// Write timeout
	writeWait = 10 * time.Second

	// Ensure a pong is received every 30 seconds
	pongWait = 30 * time.Second

	// Send a ping out every 27 seconds. Must be less than pongWait. If pong doesn't happen with pongWait - pingPeriod, the connection will timeout
	pingPeriod = (pongWait * 9) / 10
)

// ErrInvalidUsername is an error when the username does not match criteria
var ErrInvalidUsername = errors.New("server: invalid username entered")

// UsernameMaxLength is the maximum number of characters allowed in a username
const UsernameMaxLength = 25

var validUsernameRx = regexp.MustCompile(`^[\p{L}\p{M}\p{S}\p{N}\p{P} ]{1,25}\z`)
var withLetterRx = regexp.MustCompile(`\p{L}`)

// WsConn is an interface which implements a subset of the available methods in *websocket.Conn
type WsConn interface {
	Close() error
	ReadJSON(v interface{}) error
	RemoteAddr() net.Addr
	SetPongHandler(func(appDate string) error)
	SetReadDeadline(t time.Time) error
	SetReadLimit(limit int64)
	SetWriteDeadline(t time.Time) error
	WriteJSON(v interface{}) error
	WriteMessage(messageType int, data []byte) error
}

type safeIdentifier struct {
	id   int
	name string
	mu   sync.RWMutex
}

// Client represents a user connected via websocket
type Client struct {
	Game           *game.Game
	send           chan interface{}
	Conn           WsConn
	safeIdentifier safeIdentifier
}

// NewClient instantiates a new client object.
func NewClient(game *game.Game, conn WsConn, id int) *Client {
	return &Client{
		Game: game,
		send: make(chan interface{}, 256),
		Conn: conn,
		safeIdentifier: safeIdentifier{
			id:   id,
			name: name.Generate(),
		},
	}
}

// SetName sets the name of the player
func (c *Client) SetName(n string) error {
	if !validUsernameRx.MatchString(n) || !withLetterRx.MatchString(n) {
		return ErrInvalidUsername
	}

	c.safeIdentifier.mu.Lock()
	defer c.safeIdentifier.mu.Unlock()
	c.safeIdentifier.name = n
	return nil
}

// ID returns the ID of the client
func (c *Client) ID() int {
	// this won't change, no mutex necessary
	return c.safeIdentifier.id
}

// Name returns the display name for the user
func (c *Client) Name() string {
	c.safeIdentifier.mu.RLock()
	defer c.safeIdentifier.mu.RUnlock()

	return c.safeIdentifier.name
}

// Send will send an object to the client.
func (c *Client) Send(o interface{}) {
	log.Println(o)
	c.send <- o
}

// CloseChannel will close the send channel
func (c *Client) CloseChannel() {
	close(c.send)
}

// RemoteAddr returns the remote address (IP + port) of the client
func (c *Client) RemoteAddr() string {
	return c.Conn.RemoteAddr().String()
}

// WritePump writes messages to the client.
// This method should be called in a separate goroutine.
func (c *Client) WritePump(s *Server) {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(msg); err != nil {
				log.WithFields(log.Fields{"client": c.Conn.RemoteAddr().String()}).Errorf("could not write JSON: %v", err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// ReadPump reads messages sent from the client.
func (c *Client) ReadPump(s *Server) {
	defer func() {
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(readLimit)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var r WsRequest
		if err := c.Conn.ReadJSON(&r); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.WithFields(log.Fields{"client": c.Conn.RemoteAddr().String()}).Errorf("could not read JSON: %v", err)
			}
			break
		}

		s.HandleWsRequest(c, &r)
	}
}
