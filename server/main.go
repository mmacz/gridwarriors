package main

import (
	"gridwarriors/server"
	"flag"
	"log"
)



func main() {
	port := flag.Int("port", 8080, "Port to run server on")
	flag.Parse()

	if *port < 1024 || *port > 65535 {
		log.Fatalf("Invalid port number: %d\n", *port)
	}

	server.StartServer(port)
}
