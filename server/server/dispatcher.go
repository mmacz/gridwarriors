package server

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Dispatcher struct {
	sync.Mutex
	players  map[*websocket.Conn]*PlayerSession
	handlers map[string]func(*websocket.Conn, json.RawMessage)
}

type PlayerSession struct {
	Conn *websocket.Conn
	Name string
}

func NewDispatcher() *Dispatcher {
	d := &Dispatcher{
		players:  make(map[*websocket.Conn]*PlayerSession),
		handlers: make(map[string]func(*websocket.Conn, json.RawMessage)),
	}

	d.handlers["join"] = d.handleJoin
	d.handlers["leave"] = d.handleLeave

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
