package main

import (
	"encoding/json"
	"fmt"

	"ignite"
)

func main() {

	hub := ignite.NewServer(&ignite.ServerConfig{
		Namespace: "default",
		Address:   "localhost:8081",
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
		payload, _ := json.Marshal(client.ID)
		client.SendMsgExcept("#Go", []string{client.ID}, ignite.Message{
			Event:   "NEW_MEMBER",
			Payload: payload,
		})

		// Register handle for an event
		client.On("BUY", "1", func(payload json.RawMessage) {
			client.SendMsg(ignite.Message{
				Event:   "BUY_RESPONSE",
				Payload: payload,
			})

			helloMsgPayload, _ := json.Marshal("Hello world")

			// Send to all member in a room
			client.SendMsgToRoom("room-number-1", ignite.Message{
				Event:   "test_room_1",
				Payload: helloMsgPayload,
			})
		})
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.ID, " closed: ", reason)
		})
	})

	forever := make(chan struct{})
	<-forever
}
