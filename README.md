# ignite

a websocket server module.

require redis to scale to multi nodes.

use module like the code below.

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/dongnguyenltqb/ignite"
)

func main() {
	never_die := make(chan bool)
	hub := ignite.NewServer("localhost:8787", "localhost:6379", "", 10)
	hub.OnNewClient = func(client *ignite.Client) {
		client.SendIdentityMsg()
		client.On("buy", "1", func(payload json.RawMessage) {
			fmt.Println("BUY=>", string(payload))
			client.SendMsgToRoom(client.Id, ignite.Message{
				Event:   "bought",
				Payload: payload,
			})
		})
		client.On("sell", "2", func(payload json.RawMessage) {
			fmt.Println("SELL =>", string(payload))
		})
		client.On("stop_buy", "3", func(payload json.RawMessage) {
			client.Off("buy", "1")
		})
		client.OnClose(func(reason string) {
			fmt.Println("Client ", client.Id, " closed: ", reason)
		})
	}
	<-never_die
}

```

message format

```go
type Message struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
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
