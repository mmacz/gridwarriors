package server

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Dispatcher struct {
	sync.Mutex
	players  map[*websocket.Conn]*PlayerSession
	handlers map[string]func(*websocket.Conn, json.RawMessage)
	games    []*GameState
}

type PlayerSession struct {
	Conn *websocket.Conn
	Name string
	Game *GameState
}

type GameState struct {
	PlayerX    *PlayerSession
	PlayerO    *PlayerSession
	Board      [3][3]string
	Turn       string
	IsFinished bool
}

type GameStartMessage struct {
	Type string        `json:"type"`
	Data GameStartData `json:"data"`
}

type GameStartData struct {
	YourRole string `json:"your_role"`
	Opponent string `json:"opponent"`
	Turn     string `json:"turn"`
}

func NewDispatcher() *Dispatcher {
	d := &Dispatcher{
		players:  make(map[*websocket.Conn]*PlayerSession),
		games:    []*GameState{},
		handlers: make(map[string]func(*websocket.Conn, json.RawMessage)),
	}

	d.handlers["join"] = d.handleJoin
	d.handlers["leave"] = d.handleLeave
	d.handlers["start"] = d.handleStart

	return d
}

func (d *Dispatcher) HandleConnection(conn *websocket.Conn) {
	defer conn.Close()
	d.Lock()
	d.players[conn] = &PlayerSession{Conn: conn}
	d.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Disconnected: %v\n", err)
			d.handleLeave(conn, nil)
			return
		}

		var m struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg, &m); err != nil {
			log.Println("Bad message:", err)
			continue
		}

		if handler, ok := d.handlers[m.Type]; ok {
			handler(conn, m.Data)
		} else {
			log.Println("Unknown message type:", m.Type)
		}
	}
}

func (d *Dispatcher) handleJoin(conn *websocket.Conn, raw json.RawMessage) {
	var data struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Println("Invalid join data:", err)
		return
	}

	d.Lock()
	defer d.Unlock()
	if session, ok := d.players[conn]; ok {
		session.Name = data.Name
		log.Printf("Player joined: %s\n", data.Name)
	}
}

func (d *Dispatcher) handleLeave(conn *websocket.Conn, _ json.RawMessage) {
	d.Lock()
	defer d.Unlock()
	if session, ok := d.players[conn]; ok {
		log.Printf("Player left: %s\n", session.Name)
		delete(d.players, conn)
	}
}

func (d *Dispatcher) sendGameStart(p *PlayerSession, role, opponent, turn string) {
	msg := GameStartMessage{
		Type: "game_start",
		Data: GameStartData{
			YourRole: role,
			Opponent: opponent,
			Turn:     turn,
		},
	}
	if err := p.Conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send game_start to %s: %v\n", p.Name, err)
	}
}

func (d *Dispatcher) handleStart(conn *websocket.Conn, _ json.RawMessage) {
	d.Lock()
	defer d.Unlock()

	var p1, p2 *PlayerSession
	for _, p := range d.players {
		if p1 == nil {
			p1 = p
		} else if p2 == nil {
			p2 = p
			break
		}
	}

	if p1 == nil || p2 == nil {
		log.Println("Not enough players for game start")
		return
	}

	turn := "X"
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if rand.Intn(2) == 0 {
		turn = "O"
	}

	game := &GameState{
		PlayerX:    p1,
		PlayerO:    p2,
		Turn:       turn,
		IsFinished: false,
	}
	p1.Game = game
	p2.Game = game
	d.games = append(d.games, game)
	log.Printf("Game started between %s (X) and %s (O) | Turn: %s\n", p1.Name, p2.Name, turn)
	d.sendGameStart(p1, "X", p2.Name, turn)
	d.sendGameStart(p2, "O", p1.Name, turn)
}
