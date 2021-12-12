# ignite

A websocket server module.
Require redis to scale to multi nodes.

Client/server message follow format

```go
type Message struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}
```

use module like the code below.

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dongnguyenltqb/ignite"
)

func main() {
	forever := make(chan bool)

	hub := ignite.NewServer(os.Getenv("addr"), "/ws", "localhost:6379", "", 10)
	fmt.Println("Websocket is listen on", os.Getenv("addr"))
	hub.OnNewClient = func(client *ignite.Client) {
		// Send indentity message
		client.SendIdentityMsg()
		// Join a room
		client.Join("room-number-1")
		// Send to a room except some client
		client.SendMsgToRoomWithExcludeClient("room-number-1", []string{client.Id}, ignite.Message{
			Event: "new_client_enter_room",
		})
		// Register handle for an event
		client.On("buy", "1", func(payload json.RawMessage) {
			client.SendMsg(ignite.Message{
				Event:   "test",
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
			fmt.Println("Client ", client.Id, " closed: ", reason)
		})
	}
	<-forever
}

```

send/received message

```shell
➜  ignite git:(master) ✗ wscat -c "ws://localhost:8787/ws"
Connected (press CTRL+C to quit)
< {"event":"identity","payload":{"clientId":"48d44877-f04b-4e63-ad4b-ebf17899a4de"}}
> {"event":"buy","payload":"PTB"}
< {"event":"bought","payload":"PTB"}
>
```
