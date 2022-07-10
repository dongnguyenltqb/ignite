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
func main() {

	hub := ignite.NewServer[string,string](&ignite.ServerConfig{
		Namespace: "default",
		Address:   "localhost:8082",
		Path:      "/",
		RedisHost: "localhost",
		RedisPort: 6379,
		RedisDb:   10,
	})

	hub.OnNewClient(func(client *ignite.Client[string,string]) {

		// Send indentity message
		client.SendId()
		client.Set("created_at",time.Now().string())

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
```

send/received message

```shell
❯ wscat -c "ws://localhost:8081/"
Connected (press CTRL+C to quit)
< {"event":"identity","payload":{"clientId":"1ddcc65d-49ab-496a-9939-5da768d1c52c"}}
> {"event":"BUY","payload":"VCS"}
< {"event":"BUY_RESPONSE","payload":"VCS"}
>
```
