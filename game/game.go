// Package game provides capabilities for an individual estimation session.
package game

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/synacor/sibyl/deck"
)

const (
	// RoomNameValidDescription A description on how the room name should be constructed
	RoomNameValidDescription = "A room name must contain 1-20 characters with at least one being a letter or number. All characters must be letters, numbers, spaces, underscores, or hyphens"

	// RoomNameMaxLength is the max length a room name may be
	RoomNameMaxLength = 20

	// TopicMaxLength is the max length a topic may be.
	TopicMaxLength = 100
)

// waitToDestroy is the number of milliseconds to wait after last client to destroy the channel
const waitToDestroy = 10000 // 10 seconds

// ErrInvalidRoomName is returned when the room name is not valid.
var ErrInvalidRoomName = errors.New("sibyl: room name is invalid")

// Golang doesn't allow \p{Letter}, so we have to use the shorthand.
// L = Letter, M = Mark, N = Number, P = Punctuation
var validTopixRx = regexp.MustCompile(`^[\p{L}\p{M}\p{S}\p{N}\p{P} ]{1,100}\z`)
var validRoomRx = regexp.MustCompile(`^[\p{L}\p{N} _-]{1,20}\z`)
var withLetterOrNumberRx = regexp.MustCompile(`[\p{L}\p{N}]`)

type client interface {
	Send(interface{})
	ID() int
	Name() string
	CloseChannel()
	RemoteAddr() string
}

type safeClients struct {
	clients map[client]bool
	mutex   sync.RWMutex
}

type safeCards struct {
	deck   *deck.Deck
	cards  map[client]int
	reveal bool
	mutex  sync.RWMutex
}

type safeTopic struct {
	topic string
	mutex sync.RWMutex
}

type safeDestroyAttempt struct {
	attempt int
	mutex   sync.RWMutex
}

type safeClientLastID struct {
	lastID int
	mutex  sync.RWMutex
}

// Game represents an individual estimation session game
type Game struct {
	safeClients safeClients
	safeCards   safeCards
	safeTopic   safeTopic

	// Room is the name of the room
	Room string

	// Token is a unique token to ensure a user doesn't join a stale game
	Token string

	onComplete         chan *Game
	waitToDestroy      int
	safeDestroyAttempt safeDestroyAttempt
	safeClientLastID   safeClientLastID
}

type wsCard struct {
	Card     int    `json:"card"`
	PlayerID int    `json:"playerID"`
	Player   string `json:"player"`
}

// wsUpdate is an update that will be sent via websocket to the client.
type wsUpdate struct {
	Topic    string         `json:"topic"`
	Players  map[int]string `json:"players"`
	Cards    []*wsCard      `json:"cards"`
	Deck     string         `json:"deck"`
	Revealed bool           `json:"reveal"`
	Reset    bool           `json:"reset"`
	Username string         `json:"username"`
}

// wsError is providers error information to the client
type wsError struct {
	Error string `json:"error"`
}

// RoomNameIsValid validates a room name.
func RoomNameIsValid(room string) bool {
	return validRoomRx.MatchString(room) && withLetterOrNumberRx.MatchString(room)
}

// New instanties a new game.
// The onComplete chan should be used when the game is no longer active.
func New(room string, defaultDeck string, onComplete chan *Game) (*Game, error) {
	if !RoomNameIsValid(room) {
		return nil, ErrInvalidRoomName
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	useDeck := deck.ModifiedFibonacci
	if d, found := deck.AllDecks[defaultDeck]; found {
		useDeck = d
	}

	g := &Game{
		safeClients: safeClients{
			clients: make(map[client]bool),
			mutex:   sync.RWMutex{},
		},
		safeCards: safeCards{
			deck:   useDeck,
			cards:  make(map[client]int),
			reveal: false,
			mutex:  sync.RWMutex{},
		},
		safeTopic: safeTopic{
			topic: fmt.Sprintf("%s Estimation Session", room),
			mutex: sync.RWMutex{},
		},

		Room:  room,
		Token: token,

		onComplete:    onComplete,
		waitToDestroy: waitToDestroy,
		safeDestroyAttempt: safeDestroyAttempt{
			attempt: 0,
			mutex:   sync.RWMutex{},
		},
	}

	return g, nil
}

// NextClientID returns the next available ID to use for a client.
func (g *Game) NextClientID() int {
	g.safeClientLastID.mutex.Lock()
	defer g.safeClientLastID.mutex.Unlock()

	g.safeClientLastID.lastID++
	return g.safeClientLastID.lastID
}

// RegisterClient registers a client with the game.
func (g *Game) RegisterClient(client client) {
	g.safeClients.mutex.Lock()
	g.safeClients.clients[client] = true
	g.safeClients.mutex.Unlock()

	log.WithFields(log.Fields{"room": g.Room, "client": client.RemoteAddr()}).Info("registered client")

	g.SendUpdate()
}

// UnregisterClient registers a client from the game.
func (g *Game) UnregisterClient(client client) {
	g.safeClients.mutex.Lock()
	delete(g.safeClients.clients, client)
	nclients := len(g.safeClients.clients)
	g.safeClients.mutex.Unlock()

	shouldReset := false
	g.safeCards.mutex.RLock()
	_, found := g.safeCards.cards[client]
	ncards := len(g.safeCards.cards)
	g.safeCards.mutex.RUnlock()

	if found {
		g.safeCards.mutex.Lock()
		delete(g.safeCards.cards, client)
		g.safeCards.mutex.Unlock()

		// was at 1, now will be at zero. reset the game
		if ncards == 1 {
			shouldReset = true
		}
	}

	client.CloseChannel()
	log.WithFields(log.Fields{"room": g.Room, "client": client.RemoteAddr()}).Info("unregistered client")

	if nclients == 0 {
		g.reset()

		g.safeDestroyAttempt.mutex.Lock()
		g.safeDestroyAttempt.attempt++
		attempt := g.safeDestroyAttempt.attempt
		g.safeDestroyAttempt.mutex.Unlock()

		go func() {
			t := time.NewTimer(time.Millisecond * time.Duration(g.waitToDestroy))
			<-t.C

			g.safeDestroyAttempt.mutex.RLock()
			currentAttempt := g.safeDestroyAttempt.attempt
			g.safeDestroyAttempt.mutex.RUnlock()

			if attempt != currentAttempt {
				return
			}

			g.safeClients.mutex.RLock()
			defer g.safeClients.mutex.RUnlock()

			if len(g.safeClients.clients) == 0 {
				g.onComplete <- g
			}
		}()

		return
	}

	if shouldReset {
		g.Reset()
		return
	}

	g.SendUpdate()
}

// SendUpdate will send an update to all clients
func (g *Game) SendUpdate() {
	g.broadcast(g.updatePayload(false))
}

// broadcast will send a message to all registered clients.
func (g *Game) broadcast(obj interface{}) {
	g.safeClients.mutex.RLock()
	defer g.safeClients.mutex.RUnlock()

	for client := range g.safeClients.clients {
		if o, ok := obj.(wsUpdate); ok {
			o.Username = client.Name()
			obj = o
		}

		client.Send(obj)
	}
}

// errorPayload returns an object which can be sent to the client which holds an error.
func (g *Game) errorPayload(errstr string) *wsError {
	return &wsError{errstr}
}

// updatePayload returns a game update object which can be broadcasted to clients.
func (g *Game) updatePayload(reset bool) wsUpdate {
	var u wsUpdate

	g.safeCards.mutex.RLock()
	cards := make([]*wsCard, 0, len(g.safeCards.cards))
	for c, card := range g.safeCards.cards {
		cards = append(cards, &wsCard{
			Card:     card,
			Player:   c.Name(),
			PlayerID: c.ID(),
		})
	}
	u.Topic = g.Topic()
	u.Players = g.players()
	u.Deck = g.safeCards.deck.Name
	u.Cards = cards
	u.Revealed = g.safeCards.reveal
	u.Reset = reset
	g.safeCards.mutex.RUnlock()

	return u
}

func (g *Game) players() map[int]string {
	g.safeClients.mutex.RLock()
	defer g.safeClients.mutex.RUnlock()

	players := make(map[int]string)
	for client := range g.safeClients.clients {
		players[client.ID()] = client.Name()
	}

	return players
}

// SetTopic will set the topic of the room in a concurrency-safe manner.
func (g *Game) SetTopic(topic string) {
	if !validTopixRx.MatchString(topic) || !withLetterOrNumberRx.MatchString(topic) {
		return
	}

	g.safeTopic.mutex.Lock()

	didChange := false
	if topic != g.safeTopic.topic {
		didChange = true

		g.safeTopic.topic = topic
	}

	g.safeTopic.mutex.Unlock()

	if didChange {
		g.SendUpdate()
	}
}

// Topic will return the topic of the room in a concurrency-safe manner.
func (g *Game) Topic() string {
	g.safeTopic.mutex.RLock()
	defer g.safeTopic.mutex.RUnlock()

	return g.safeTopic.topic
}

// SetDeck changes the active deck being used.
func (g *Game) SetDeck(deck *deck.Deck) {
	g.safeCards.mutex.Lock()
	if deck == g.safeCards.deck {
		g.safeCards.mutex.Unlock()
		return
	}
	g.safeCards.deck = deck
	g.safeCards.mutex.Unlock()

	g.Reset()
}

// Deck returns the active deck being used.
func (g *Game) Deck() *deck.Deck {
	g.safeCards.mutex.RLock()
	defer g.safeCards.mutex.RUnlock()
	return g.safeCards.deck
}

// AddCard is when a client has selected an individual card.
func (g *Game) AddCard(c client, card int, deck string) {
	g.safeCards.mutex.Lock()

	if deck != g.safeCards.deck.Name {
		log.WithFields(log.Fields{"room": g.Room, "client": c.RemoteAddr()}).Warnf("client is out of sync: got %s, expects %s", deck, g.safeCards.deck.Name)
		c.Send(g.errorPayload("Your game is out of sync. Please refresh your browser."))
		g.safeCards.mutex.Unlock()
		return
	}

	if _, err := g.safeCards.deck.GetCard(card); err != nil {
		log.WithFields(log.Fields{"room": g.Room, "client": c.RemoteAddr()}).Warnf("client submitted an invalid card (%d) for deck \"%s\"", card, g.safeCards.deck.Name)
		c.Send(g.errorPayload("Your game had an invalid card. Please refresh your browser."))
		g.safeCards.mutex.Unlock()
		return
	}

	g.safeCards.cards[c] = card
	ncards := len(g.safeCards.cards)
	g.safeCards.mutex.Unlock()

	g.safeClients.mutex.RLock()
	nclients := len(g.safeClients.clients)
	g.safeClients.mutex.RUnlock()

	if ncards == nclients {
		g.safeCards.mutex.Lock()
		g.safeCards.reveal = true
		g.safeCards.mutex.Unlock()
	}

	g.SendUpdate()
}

// Reveal is when a client has requested to show all the cards.
func (g *Game) Reveal() {
	g.safeCards.mutex.Lock()
	g.safeCards.reveal = true
	g.safeCards.mutex.Unlock()

	g.SendUpdate()
}

// Reset is when a client has request that the entire game be reset.
func (g *Game) Reset() {
	g.reset()
	g.broadcast(g.updatePayload(true))
}

func (g *Game) reset() {
	g.safeCards.mutex.Lock()
	g.safeCards.reveal = false
	g.safeCards.cards = make(map[client]int)
	g.safeCards.mutex.Unlock()
}

// RegisteredClientsCount returns the number of active registered clients
func (g *Game) RegisteredClientsCount() int {
	g.safeClients.mutex.RLock()
	defer g.safeClients.mutex.RUnlock()

	return len(g.safeClients.clients)
}

func generateToken() (string, error) {
	b := make([]byte, 30)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}
