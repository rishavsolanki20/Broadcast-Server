//client.go

package client

import (
	"bufio"
	"log"
	"os"

	"github.com/gorilla/websocket"
)

func ConnectToServer(serverURL string) {
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatalf("Error connecting to server: %v\n", err)
	}
	defer conn.Close()

	log.Println("Connected to the server. Waiting for updates...")

	go func() {
		// Read incoming messages from the server
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Disconnected from server")
				return
			}
			log.Printf("Received update: %s\n", string(msg)) // Log the update to check if messages are received
		}

	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := scanner.Text()
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Println("Error sending message: ", err)
			return
		}
		log.Println("Sent update:", msg)
	}

}
