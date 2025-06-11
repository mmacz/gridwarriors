package main

import "gridwarriors/server"

func main() {
	port := 8080
	server.StartServer(&port)
}
