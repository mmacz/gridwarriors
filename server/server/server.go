package server

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func (r* http.Request) bool { return true },
}

var dispatcher = NewDispatcher()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	dispatcher.HandleConnection(conn)
}

func StartServer(port *int) {
	addr := fmt.Sprintf("localhost:%d", *port)
	log.Printf("Server starting: %s\n", addr)
	http.HandleFunc("/ws", wsHandler)
	log.Fatalln(http.ListenAndServe(addr, nil))
}
