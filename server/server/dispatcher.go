package server

import (
	"encoding/json"
	"fmt"
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
	ID         string
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

type GameMoveMessage struct {
	Type string       `json:"type"`
	Data GameMoveData `json:"data"`
}

type GameStartData struct {
	GameID   string `json:"game_id"`
	YourRole string `json:"your_role"`
	Opponent string `json:"opponent"`
	Turn     string `json:"turn"`
}

type GameMoveData struct {
	X int `json:"x"`
	Y int `json:"y"`
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
	d.handlers["move"] = d.handleMove

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
		conn.Close()
	}
}

func (d *Dispatcher) sendGameStart(p *PlayerSession, game *GameState, role, opponent, turn string) {
	msg := GameStartMessage{
		Type: "game_start",
		Data: GameStartData{
			GameID:   game.ID,
			YourRole: role,
			Opponent: opponent,
			Turn:     turn,
		},
	}
	if err := p.Conn.WriteJSON(msg); err != nil {
		log.Printf("[Game %s] Failed to send game_start to %s: %v\n", game.ID, p.Name, err)
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

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	turn := "X"
	if r.Intn(2) == 0 {
		turn = "O"
	}

	game := &GameState{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		PlayerX:    p1,
		PlayerO:    p2,
		Turn:       turn,
		IsFinished: false,
	}
	p1.Game = game
	p2.Game = game
	d.games = append(d.games, game)

	log.Printf("[Game %s] Started between %s (X) and %s (O) | Turn: %s\n", game.ID, p1.Name, p2.Name, turn)
	d.sendGameStart(p1, game, "X", p2.Name, turn)
	d.sendGameStart(p2, game, "O", p1.Name, turn)
}

func (g *GameState) GetRole(p *PlayerSession) (string, bool) {
	switch p {
	case g.PlayerX:
		return "X", true
	case g.PlayerO:
		return "O", true
	default:
		return "", false
	}
}

func (d *Dispatcher) sendError(p *PlayerSession, message string) {
	errMsg := map[string]interface{}{
		"type": "error",
		"data": map[string]string{
			"message": message,
		},
	}
	if err := p.Conn.WriteJSON(errMsg); err != nil {
		log.Printf("Failed to send error to %s: %v\n", p.Name, err)
	}
}

func (d *Dispatcher) sendUpdate(g *GameState) {
	msg := map[string]interface{}{
		"type": "game_update",
		"data": map[string]interface{}{
			"board": g.Board,
			"turn":  g.Turn,
		},
	}
	if err := g.PlayerX.Conn.WriteJSON(msg); err != nil {
		log.Printf("[Game %s] Failed to send update to %s: %v\n", g.ID, g.PlayerX.Name, err)
	}
	if err := g.PlayerO.Conn.WriteJSON(msg); err != nil {
		log.Printf("[Game %s] Failed to send update to %s: %v\n", g.ID, g.PlayerO.Name, err)
	}
}

func CheckWinner(board [3][3]string) (string, bool) {
	lines := [8][3][2]int{
		{{0, 0}, {0, 1}, {0, 2}},
		{{1, 0}, {1, 1}, {1, 2}},
		{{2, 0}, {2, 1}, {2, 2}},
		{{0, 0}, {1, 0}, {2, 0}},
		{{0, 1}, {1, 1}, {2, 1}},
		{{0, 2}, {1, 2}, {2, 2}},
		{{0, 0}, {1, 1}, {2, 2}},
		{{0, 2}, {1, 1}, {2, 0}},
	}
	for _, line := range lines {
		a, b, c := line[0], line[1], line[2]
		if board[a[0]][a[1]] != "" &&
			board[a[0]][a[1]] == board[b[0]][b[1]] &&
			board[a[0]][a[1]] == board[c[0]][c[1]] {
			return board[a[0]][a[1]], false
		}
	}

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i][j] == "" {
				return "", false
			}
		}
	}
	fmt.Println("BOARD FULL — DRAW DETECTED")
	return "", true
}

func buildGameEndMessage(winner, role string, draw bool) map[string]interface{} {
	result := "lose"
	winnerField := winner

	if draw {
		result = "draw"
		winnerField = ""
	} else if role == winner {
		result = "win"
	}

	return map[string]interface{}{
		"type": "game_end",
		"data": map[string]string{
			"winner": winnerField,
			"result": result,
		},
	}
}

func (d *Dispatcher) sendGameEnd(game *GameState, winner string, draw bool) {
	for _, player := range []*PlayerSession{game.PlayerX, game.PlayerO} {
		role, _ := game.GetRole(player)
		msg := buildGameEndMessage(winner, role, draw)

		if err := player.Conn.WriteJSON(msg); err != nil {
			log.Printf("[Game %s] Failed to send game_end to %s: %v\n", game.ID, player.Name, err)
		}
	}
}

func formatBoard(b [3][3]string) string {
	var out string
	for _, row := range b {
		for _, cell := range row {
			if cell == "" {
				out += "."
			} else {
				out += cell
			}
		}
		out += "\n"
	}
	return out
}

func (d *Dispatcher) handleMove(conn *websocket.Conn, raw json.RawMessage) {
	d.Lock()
	defer d.Unlock()

	p, ok := d.players[conn]
	if !ok || p == nil || p.Game == nil || p.Game.IsFinished {
		log.Printf("Invalid move: no session or game inactive (conn: %v)\n", conn.RemoteAddr())
		if ok && p != nil {
			d.sendError(p, "Invalid game state")
		}
		return
	}

	game := p.Game

	var move GameMoveData
	if err := json.Unmarshal(raw, &move); err != nil {
		log.Printf("[Game %s] Invalid move data from %s: %v\n", game.ID, p.Name, err)
		d.sendError(p, "Invalid move data")
		return
	}

	role, ok := game.GetRole(p)
	if !ok {
		log.Printf("[Game %s] Player %s not part of the game\n", game.ID, p.Name)
		d.sendError(p, "You are not part of this game")
		return
	}

	if game.Turn != role {
		log.Printf("[Game %s] Not %s's turn (you are %s, current turn: %s)\n", game.ID, p.Name, role, game.Turn)
		d.sendError(p, "It's not your turn")
		return
	}

	if move.X < 0 || move.X >= 3 || move.Y < 0 || move.Y >= 3 {
		log.Printf("[Game %s] Invalid move from %s: x=%d y=%d\n", game.ID, p.Name, move.X, move.Y)
		d.sendError(p, "Invalid move coordinates")
		return
	}

	if game.Board[move.Y][move.X] != "" {
		log.Printf("[Game %s] Cell already occupied at (%d,%d) by %s\n", game.ID, move.X, move.Y, game.Board[move.Y][move.X])
		d.sendError(p, "Cell already occupied")
		return
	}

	game.Board[move.Y][move.X] = role
	log.Printf("[Game %s] %s (%s) played at (%d,%d)\n", game.ID, p.Name, role, move.X, move.Y)

	log.Printf("[Game %s] Current board state:\n%s", game.ID, formatBoard(game.Board))
	winner, draw := CheckWinner(game.Board)
	if winner != "" || draw {
		game.IsFinished = true
		d.sendGameEnd(game, winner, draw)
		return
	}

	game.Turn = map[string]string{"X": "O", "O": "X"}[game.Turn]
	d.sendUpdate(game)
}
