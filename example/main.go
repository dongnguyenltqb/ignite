package main

import (
	"encoding/json"
	"fmt"
	"os"

	"ignite"
)

func main() {
	forever := make(chan bool)

	hub := ignite.NewServer(os.Getenv("addr"), "localhost:6379", "", 10)
	hub.OnNewClient = func(client *ignite.Client) {
		client.SendIdentityMsg()
		client.Join("room-number-1")
		client.On("buy", "1", func(payload json.RawMessage) {
			client.SendMessage(ignite.Message{
				Event:   "test",
				Payload: payload,
			})
			b, _ := json.Marshal("Hello world")
			client.SendMsgToRoom("room-number-1", ignite.Message{
				Event:   "test_room_1",
				Payload: b,
			})
		})
		client.On("stop_buy", "3", func(payload json.RawMessage) {
			client.Off("buy", "1")
			client.Leave("room-number-1")

		})
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.Id, " closed: ", reason)
		})
	}

	<-forever
}
