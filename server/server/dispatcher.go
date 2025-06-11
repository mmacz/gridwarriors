package server

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type PlayerSession struct {
	Conn *websocket.Conn
	Name string
}

type Dispatcher struct {
	sync.Mutex
	players map[*websocket.Conn]*PlayerSession
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher {
		players: make(map[*websocket.Conn]*PlayerSession),
	}
}

func (d* Dispatcher) HandleConnection(conn* websocket.Conn) {
	defer conn.Close()
	d.Lock()
	d.players[conn] = &PlayerSession{Conn: conn}
	d.Unlock()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Disconnected: %v\n", err)
			d.handleLeave(conn)
		}

		var m struct {
			Type string	`json:"type"`
			Data json.RawMessage `json:"data"`
		}

		if err := json.Unmarshal(msg, &m); err != nil {
			log.Printf("Bad message: ", err)
			continue
		}

		switch m.Type {
		case "join":
			var data struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(m.Data, &data); err == nil {
				d.handleJoin(conn, data.Name)
			}
		case "leave":
			d.handleLeave(conn)
		default:
			log.Println("Unknown message type: ", m.Type)
		}
	}
}

func (d* Dispatcher) handleJoin(conn* websocket.Conn, name string) {
	d.Lock()
	defer d.Unlock()

	if session, ok := d.players[conn]; ok {
		session.Name = name
		log.Printf("Player joined: %s\n", name)
	}
}

func (d* Dispatcher) handleLeave(conn* websocket.Conn) {
	d.Lock()
	defer d.Unlock()

	if session, ok := d.players[conn]; ok {
		log.Printf("Player left: %s\n", session.Name)
		delete(d.players, conn)
	}
}

