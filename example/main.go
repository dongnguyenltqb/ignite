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
		newMemberPayload, _ := json.Marshal(client.Id)
		client.BroadcastMsg(ignite.Message{
			Event:   "new_member",
			Payload: newMemberPayload,
		})
		client.On("buy", "1", func(payload json.RawMessage) {
			client.SendMessage(ignite.Message{
				Event:   "test",
				Payload: payload,
			})
			helloMsgPayload, _ := json.Marshal("Hello world")
			client.SendMsgToRoom("room-number-1", ignite.Message{
				Event:   "test_room_1",
				Payload: helloMsgPayload,
			})
		})
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.Id, " closed: ", reason)
		})
	}

	<-forever
}
