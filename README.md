# ignite

websocket server module
require redis to scale to multi node
use module like the code below.

```go
	package main

	import (
		"encoding/json"
		"fmt"
		"time"

		"github.com/dongnguyenltqb/ignite"
	)

	func main() {
		never_die := make(chan bool)
		hub := ignite.NewServer("localhost:8787", "localhost:6379", "", 10)
		hub.OnNewClient = func(client *ignite.Client) {
			client.SendIdentityMsg()
			client.On("buy", "1", func(raw json.RawMessage) {
				fmt.Println("BUY=>", string(raw))
			})
			client.On("sell", "2", func(raw json.RawMessage) {
				fmt.Println("SELL =>", string(raw))
			})
			client.On("stop_buy", "3", func(raw json.RawMessage) {
				client.Off("buy", "1")
			})
			client.OnClose(func(reason string) {
				fmt.Println("Client ", client.Id, " closed: ", reason)
			})
			<-time.After(time.Second)
			client.SendMsgToRoom(client.Id, []byte("Hello"))
		}
		<-never_die
	}
```
