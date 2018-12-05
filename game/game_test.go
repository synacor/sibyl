package game

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synacor/sibyl/deck"
)

func TestRoomNameIsValid(t *testing.T) {
	assert.True(t, RoomNameIsValid("Test Room"))
	assert.True(t, RoomNameIsValid("ÉÉÉÉÉÉÉÉÉÉÉÉÉÉÉÉÉÉÉÉ"))
	assert.False(t, RoomNameIsValid("Room name is too long"))
}

func TestGame(t *testing.T) {
	g, err := New("Room name is too long", "", nil)
	assert.Nil(t, g)
	assert.Equal(t, ErrInvalidRoomName, err)

	g, _ = New("Test Room", "bad", nil)
	assert.Equal(t, "Test Room", g.Room)
	assert.Equal(t, 40, len(g.Token))
	assert.Equal(t, deck.ModifiedFibonacci, g.Deck())
	assert.Equal(t, 0, g.RegisteredClientsCount())

	g, _ = New("Test Room", "T-Shirt Sizes", nil)
	assert.Equal(t, deck.TShirtSizes, g.Deck())
}

func TestRegisterClients(t *testing.T) {
	g, _ := New("Test", "", nil)

	c1, c2 := newClientTest(1), newClientTest(2)

	g.RegisterClient(c1)
	g.RegisterClient(c2)
	assert.Equal(t, 2, g.RegisteredClientsCount())

	assert.Equal(t, 2, len(c1.send))
	assert.Equal(t, 1, len(c2.send))

	send := append(c1.send, c2.send...)
	for _, s := range send {
		u := s.(wsUpdate)
		assert.Equal(t, "Test Estimation Session", u.Topic)
		assert.Equal(t, deck.ModifiedFibonacci.Name, u.Deck)
		assert.Equal(t, false, u.Reset)
		assert.Equal(t, false, u.Revealed)
		assert.Equal(t, []*wsCard{}, u.Cards)
	}

	assert.Equal(t, 1, len(send[0].(wsUpdate).Players))
	assert.Equal(t, 2, len(send[1].(wsUpdate).Players))
	assert.Equal(t, 2, len(send[2].(wsUpdate).Players))
}

func TestAddCard(t *testing.T) {
	g, _ := New("Test", "", nil)
	c1, c2 := newClientTest(1), newClientTest(2)
	g.RegisterClient(c1)
	g.RegisterClient(c2)

	g.AddCard(c1, 0, g.Deck().Name)
	g.AddCard(c1, 1, g.Deck().Name)
	g.AddCard(c2, 2, g.Deck().Name)

	// c1 = 2 client registers + 3 cards
	// c2 = 1 client registers + 3 cards
	assert.Equal(t, 5, len(c1.send))
	assert.Equal(t, 4, len(c2.send))

	u := c2.send[3].(wsUpdate)
	sort.Sort(byID(u.Cards))

	assert.Equal(t, []*wsCard{
		{1, 1, ""},
		{2, 2, ""},
	}, u.Cards)

	assert.Equal(t, false, u.Reset)

	// only one card sent
	assert.Equal(t, false, c2.send[1].(wsUpdate).Revealed)
	// two cards sent, but both from same client
	assert.Equal(t, false, c2.send[2].(wsUpdate).Revealed)
	// all clients sent a card, reveal
	assert.Equal(t, true, u.Revealed)
}

func TestAddCardWithOutOfSyncDeck(t *testing.T) {
	g, _ := New("Test", "", nil)
	c1, c2 := newClientTest(1), newClientTest(2)
	g.RegisterClient(c1)
	g.RegisterClient(c2)

	g.AddCard(c1, 0, "Bad")
	assert.Equal(t, 3, len(c1.send)) // 2 reg + 1 error
	assert.Equal(t, 1, len(c2.send)) // 1 reg
	assert.Equal(t, "Your game is out of sync. Please refresh your browser.", c1.send[2].(*wsError).Error)
}

func TestAddCardWithIncorrectCard(t *testing.T) {
	g, _ := New("Test", "", nil)
	c1, c2 := newClientTest(1), newClientTest(2)
	g.RegisterClient(c1)
	g.RegisterClient(c2)

	g.AddCard(c1, 9999, "Modified Fibonacci")
	assert.Equal(t, 3, len(c1.send)) // 2 reg + 1 error
	assert.Equal(t, 1, len(c2.send)) // 1 reg
	assert.Equal(t, "Your game had an invalid card. Please refresh your browser.", c1.send[2].(*wsError).Error)
}

func TestReveal(t *testing.T) {
	g, _ := New("Test", "", nil)
	c1 := newClientTest(1)
	g.RegisterClient(c1)

	g.Reveal()

	assert.Equal(t, 2, len(c1.send))

	u := c1.send[1].(wsUpdate)
	assert.Equal(t, true, u.Revealed)
	assert.Equal(t, false, u.Reset)
}

func TestReset(t *testing.T) {
	g, _ := New("Test", "", nil)
	g.safeCards.reveal = true
	g.safeCards.cards = map[client]int{newClientTest(1): 0, newClientTest(2): 1, newClientTest(3): 2}

	c1 := newClientTest(1)
	g.RegisterClient(c1)

	assert.True(t, time.Now().After(g.safeClock.clock))
	clock := g.safeClock.clock

	g.Reset()

	// make sure clock is updated on reset
	assert.True(t, time.Now().After(g.safeClock.clock))
	assert.True(t, clock.Before(g.safeClock.clock))

	assert.Equal(t, 2, len(c1.send))

	u1 := c1.send[0].(wsUpdate)
	assert.Equal(t, true, u1.Revealed)
	assert.Equal(t, false, u1.Reset)
	sort.Sort(byID(u1.Cards))
	assert.Equal(t, []*wsCard{
		{0, 1, ""},
		{1, 2, ""},
		{2, 3, ""},
	}, u1.Cards)

	u2 := c1.send[1].(wsUpdate)
	assert.Equal(t, false, u2.Revealed)
	assert.Equal(t, true, u2.Reset)
	assert.Equal(t, []*wsCard{}, u2.Cards)
}

func TestSetDeck(t *testing.T) {
	g, _ := New("Test", "", nil)
	c1 := newClientTest(1)

	g.RegisterClient(c1)

	// default, so it shouldn't change
	g.SetDeck(deck.ModifiedFibonacci)
	assert.Equal(t, deck.ModifiedFibonacci, g.Deck())
	assert.Equal(t, 1, len(c1.send))

	g.SetDeck(deck.TShirtSizes)
	assert.Equal(t, deck.TShirtSizes.Name, g.Deck().Name)
	assert.Equal(t, 2, len(c1.send))
	assert.Equal(t, deck.TShirtSizes.Name, c1.send[1].(wsUpdate).Deck)
}

func TestSetTopic(t *testing.T) {
	g, _ := New("Test", "", nil)
	c1 := newClientTest(1)
	g.RegisterClient(c1)

	g.SetTopic("Should be invalid: \t")
	assert.Equal(t, "Test Estimation Session", g.Topic())
	assert.Equal(t, 1, len(c1.send))

	g.SetTopic(strings.Repeat("É", 101))
	assert.Equal(t, "Test Estimation Session", g.Topic())
	assert.Equal(t, 1, len(c1.send))

	g.SetTopic("New Topic")
	assert.Equal(t, "New Topic", g.Topic())
	assert.Equal(t, 2, len(c1.send))
	assert.Equal(t, "New Topic", c1.send[1].(wsUpdate).Topic)
}

func TestUnregisterClient(t *testing.T) {
	onComplete := make(chan *Game)

	g, _ := New("Test", "", onComplete)
	g.waitToDestroy = 1

	c1, c2 := newClientTest(1), newClientTest(2)
	g.RegisterClient(c1)
	g.RegisterClient(c2)

	// make sure everything is in good state first
	assert.Equal(t, 2, len(c1.send))
	assert.Equal(t, 1, len(c2.send))

	g.UnregisterClient(c2)
	assert.Equal(t, 3, len(c1.send))
	assert.Equal(t, 1, len(c2.send))

	timer := time.NewTimer(time.Millisecond * time.Duration(5))
	select {
	case <-onComplete:
		assert.Fail(t, "should not have been hit")
	case <-timer.C:
	}

	g.UnregisterClient(c1)
	assert.Equal(t, 3, len(c1.send))
	assert.Equal(t, 1, len(c2.send))

	timer = time.NewTimer(time.Millisecond * time.Duration(5))
	select {
	case g2 := <-onComplete:
		assert.Equal(t, g, g2)
	case <-timer.C:
		assert.Fail(t, "should not have been hit")
	}
	timer.Stop()
}

func TestNextClientID(t *testing.T) {
	g, _ := New("Test", "", nil)
	assert.Equal(t, 1, g.NextClientID())
	assert.Equal(t, 2, g.NextClientID())
	assert.Equal(t, 3, g.NextClientID())
}

type clientTest struct {
	send                []interface{}
	closeChannelInvoked int
	port                int
	id                  int
	name                string
}

func newClientTest(id int) *clientTest {
	return &clientTest{
		send: make([]interface{}, 0),
		port: id,
		id:   id,
	}
}

func (c *clientTest) Send(o interface{}) {
	c.send = append(c.send, o)
}

func (c *clientTest) CloseChannel() {
	c.closeChannelInvoked++
}

func (c *clientTest) RemoteAddr() string {
	return fmt.Sprintf("1.2.3.4:%d", c.port)
}

func (c *clientTest) ID() int {
	return c.id
}

func (c *clientTest) Name() string {
	return c.name
}

type byID []*wsCard

func (b byID) Len() int           { return len(b) }
func (b byID) Less(i, j int) bool { return b[i].PlayerID < b[j].PlayerID }
func (b byID) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
