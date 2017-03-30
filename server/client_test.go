package server

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synacor/sibyl/game"
)

var panicError string

func TestSendAndClose(t *testing.T) {
	g, _ := game.New("Test", "", nil)
	c := NewClient(g, nil, 1)
	c.Send("Test")
	a := <-c.send
	assert.Equal(t, "Test", a.(string))

	c.CloseChannel()
	capturePanic(func() {
		c.send <- true
	})
	assert.Equal(t, "send on closed channel", panicError)
}

func TestID(t *testing.T) {
	c := NewClient(&game.Game{}, newWsConn(), 5)
	assert.Equal(t, 5, c.ID())
}

func TestName(t *testing.T) {
	c := NewClient(&game.Game{}, newWsConn(), 5)
	s := strings.Split(c.Name(), " ")
	assert.Equal(t, 2, len(s))
	assert.Regexp(t, `^[A-Z][a-z]+\z`, s[0])
	assert.Regexp(t, `^[A-Z][a-z]+\z`, s[1])

	err := c.SetName("Göod Name!")
	assert.NoError(t, err)
	assert.Equal(t, "Göod Name!", c.Name())

	err = c.SetName("!!!")
	assert.Equal(t, ErrInvalidUsername, err)

	assert.Equal(t, 25, UsernameMaxLength)

	name := strings.Repeat("a", UsernameMaxLength)
	err = c.SetName(name)
	assert.NoError(t, err)
	assert.Equal(t, name, c.Name())

	err = c.SetName(strings.Repeat("a", UsernameMaxLength+1))
	assert.Equal(t, ErrInvalidUsername, err)
}

func TestRemoteAddr(t *testing.T) {
	conn := newWsConn()
	conn.addr = &addr{"1.2.3.4"}
	g, _ := game.New("Test", "", nil)

	c := NewClient(g, conn, 1)
	assert.Equal(t, "1.2.3.4", c.RemoteAddr())
}

func TestWritePump(t *testing.T) {
	g, _ := game.New("Test", "", nil)
	conn := newWsConn()
	c := NewClient(g, conn, 1)

	go func() {
		c.send <- "Test"
		c.CloseChannel()
	}()

	c.WritePump(nil)

	assert.Equal(t, 2, len(conn.writeDeadline))
	assert.True(t, conn.writeDeadline[1].After(conn.writeDeadline[0]))
	assert.True(t, conn.writeDeadline[1].After(time.Now()))
	assert.Equal(t, "Test", conn.writeJSON.(string))
}

type wsConn struct {
	addr             *addr
	closeInvoked     int
	writeDeadline    []time.Time
	writeMessageType int
	writeMessageData []byte
	writeJSON        interface{}
}

func newWsConn() *wsConn {
	return &wsConn{
		writeDeadline: make([]time.Time, 0),
	}
}

type addr struct{ ip string }

func (a *addr) Network() string {
	return ""
}

func (a *addr) String() string {
	return a.ip
}

func (c *wsConn) Close() error                              { c.closeInvoked++; return nil }
func (c *wsConn) ReadJSON(v interface{}) error              { return nil }
func (c *wsConn) RemoteAddr() net.Addr                      { return c.addr }
func (c *wsConn) SetPongHandler(func(appDate string) error) {}
func (c *wsConn) SetReadDeadline(t time.Time) error         { return nil }
func (c *wsConn) SetReadLimit(limit int64)                  {}
func (c *wsConn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = append(c.writeDeadline, t)
	return nil
}
func (c *wsConn) WriteJSON(v interface{}) error { c.writeJSON = v; return nil }
func (c *wsConn) WriteMessage(messageType int, data []byte) error {
	c.writeMessageType = messageType
	c.writeMessageData = data
	return nil
}

func capturePanic(fn func()) {
	panicError = ""

	defer func() {
		if r := recover(); r != nil {
			panicError = r.(error).Error()
		}
	}()

	fn()
}
