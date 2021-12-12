# ignite

a websocket server module.

require redis to scale to multi nodes.

client/server message format

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

	"ignite"
)

func main() {
	forever := make(chan bool)

	hub := ignite.NewServer(os.Getenv("addr"), "/ws", "localhost:6379", "", 10)
	hub.OnNewClient = func(client *ignite.Client) {
		// Send identity message, include client id
		client.SendIdentityMsg()
		// Join a room
		client.Join("room-number-1")
		newMemberPayload, _ := json.Marshal(client.Id)
		// Broadcast
		client.BroadcastMsg(ignite.Message{
			Event:   "new_member",
			Payload: newMemberPayload,
		})
		// Listen an event
		client.On("buy", "1", func(payload json.RawMessage) {
			// Send message
			client.SendMessage(ignite.Message{
				Event:   "test",
				Payload: payload,
			})
			helloMsgPayload, _ := json.Marshal("Hello world")
			// Send message to room
			client.SendMsgToRoom("room-number-1", ignite.Message{
				Event:   "test_room_1",
				Payload: helloMsgPayload,
			})
		})
		// Handle close connection event
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
