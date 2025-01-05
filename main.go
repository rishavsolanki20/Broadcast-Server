//main.go

package main

import (
	"flag"
	"fmt"
	"github.com/rishavsolanki20/Broadcast-Server/client"
	"strings"
)

func main() {

	mode := flag.String("mode", "start", "Mode: start (server) or connect (client)")
	port := flag.String("port", "8080", "Port for the server")
	peers := flag.String("peers", "", "Comma-separated list of peer server addresses")
	serverURL := flag.String("server", "ws://localhost:8080/ws", "Server WebSocket URL")

	flag.Parse()

	switch *mode {
	case "start":
		server := NewServer()
		peerList := []string{}
		if *peers != "" {
			peerList = strings.Split(*peers, ",")
		}
		server.Start(*port, peerList)
	case "connect":
		ConnectToServer(*serverURL)
	default:
		fmt.Println("Invalid mode. Use 'start' to start the server or 'connect' to connect as a client.")
	}
}
