# ignite
websocket server module


package main

import (
	"encoding/json"
	"fmt"
	"ignite"
)

func main() {
	never_die := make(chan bool)
	hub := ignite.NewServer(8787)
	hub.OnNewClient = func(client *ignite.Client) {
		client.On("buy", "1", func(raw json.RawMessage) {
			fmt.Println("BUY=>", string(raw))
		})
		client.On("sell", "2", func(raw json.RawMessage) {
			fmt.Println("SELL =>", string(raw))
		})
		client.On("stop_buy", "3", func(raw json.RawMessage) {
			client.Off("buy", "1")
		})
	}
	<-never_die
}
