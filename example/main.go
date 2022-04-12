package main

import (
	"encoding/json"
	"fmt"

	"ignite"
)

func main() {

	hub := ignite.NewServer(&ignite.ServerConfig{
		Namespace: "default",
		Address:   "localhost:8082",
		Path:      "/",
		RedisHost: "localhost",
		RedisPort: 6379,
		RedisDb:   10,
	})

	hub.OnNewClient(func(client *ignite.Client) {

		// Send indentity message
		client.SendId()

		// Join a room
		client.Join("#Go")

		// Send to a room except some client
		client.SendMsgExcept("#Go", []string{client.ID}, ignite.Message{
			Event:   "NEW_MEMBER",
			Payload: ignite.ToPayload(client.ID),
		})

		// Register handle for an event
		client.On("BUY", "1", func(payload json.RawMessage) {
			client.SendMsg(ignite.Message{
				Event:   "BUY_RESPONSE",
				Payload: payload,
			})

		})
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.ID, " closed: ", reason)
		})
	})

	forever := make(chan struct{})
	<-forever
}
