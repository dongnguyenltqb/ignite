package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dongnguyenltqb/ignite"
)

func main() {
	forever := make(chan bool)
	// Server 1 listen on addr
	hub := ignite.NewServer(os.Getenv("addr"), "localhost:6379", "", 10)
	hub.OnNewClient = func(client *ignite.Client) {
		client.SendIdentityMsg()
		client.Join("room-hub")
		client.On("buy", "1", func(payload json.RawMessage) {
			client.SendMessage(ignite.Message{
				Event:   "test",
				Payload: payload,
			})
			client.SendMsgToRoom("room-1", ignite.Message{
				Event:   "test_room_1",
				Payload: payload,
			})
		})
		client.On("stop_buy", "3", func(payload json.RawMessage) {
			client.Off("buy", "1")
		})
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.Id, " closed: ", reason)
		})
	}
	//	server 2 listen on addr2
	leo := ignite.NewServer(os.Getenv("addr2"), "localhost:6379", "", 10)
	leo.OnNewClient = func(client *ignite.Client) {
		client.SendIdentityMsg()
		client.Join("room-leo")
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.Id, " closed: ", reason)
		})
	}
	<-forever
}
