package ignite

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// The Subscribe channel for room message
	pubSubRoomChannel = "ignite_room_chan_"
	// The Subscribe channel for broadcast message
	pubSubBroadcastChannel = "ignite_broadcast_chan_"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 	serveWs handles websocket requests from the peer.
func (h *Hub[K, V]) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client[K, V]{
		ID:                 uuid.New().String(),
		hub:                h,
		conn:               conn,
		send:               make(chan []byte, 256),
		rchan:              make(chan wsRoomActionMessage, 100),
		rooms:              make([]string, maxRoomSize),
		onCloseHandelFuncs: []func(string){},
		logger:             getLogger(),
		metadata:           make(map[K]V),
	}
	client.rooms = []string{client.ID}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.roomPump()
	go client.writePump()
	go client.readPump()
	for _, handler := range h.handleFn {
		handler(client)
	}

}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub[K comparable, V any] struct {
	// Namespace
	namespace string
	// Node Id
	nodeId string
	// Registered clients.
	clients map[*Client[K, V]]bool

	// Outbound message for specific client
	directMsg chan wsDirectMessage[K, V]

	// Outbound messages
	broadcast chan []byte

	// Outbound message for a specific room
	room chan wsMessageForRoom

	// Register requests from the clients.
	register chan *Client[K, V]

	// Unregister requests from clients.
	unregister chan *Client[K, V]

	// Pubsub channel
	pubSubRoomChannel      string
	pubSubBroadcastChannel string
	subscribeRoomChan      <-chan *redis.Message
	subscribeBroadcastChan <-chan *redis.Message

	// Logger
	logger *logrus.Logger

	// HandleFunc
	handleFn []func(client *Client[K, V])
}

func (h *Hub[K, V]) OnNewClient(fn func(client *Client[K, V])) {
	h.handleFn = append(h.handleFn, fn)
}

func newHub[K comparable, V any](namespace string) *Hub[K, V] {
	redisSubscribeRoom := getRedis().Subscribe(context.Background(), pubSubRoomChannel+namespace)
	redisSubscribeBroadcast := getRedis().Subscribe(context.Background(), pubSubBroadcastChannel+namespace)
	hub := &Hub[K, V]{
		namespace:              namespace,
		nodeId:                 uuid.New().String(),
		directMsg:              make(chan wsDirectMessage[K, V]),
		broadcast:              make(chan []byte),
		room:                   make(chan wsMessageForRoom),
		register:               make(chan *Client[K, V]),
		unregister:             make(chan *Client[K, V]),
		clients:                make(map[*Client[K, V]]bool),
		pubSubRoomChannel:      pubSubRoomChannel,
		pubSubBroadcastChannel: pubSubBroadcastChannel,
		subscribeRoomChan:      redisSubscribeRoom.Channel(),
		subscribeBroadcastChan: redisSubscribeBroadcast.Channel(),
		logger:                 getLogger(),
		handleFn:               make([]func(*Client[K, V]), 0),
	}
	go hub.run()
	return hub
}

// Send message to specific room
func (h *Hub[K, V]) SendMsgToRoom(roomId string, message Message) {
	b, err := json.Marshal(message)
	if err != nil {
		h.logger.Error(err)
	}
	h.room <- wsMessageForRoom{
		NodeId:  h.nodeId,
		RoomId:  roomId,
		Message: b,
	}
}

// Send message to specific room except some client
func (h *Hub[K, V]) SendMsgExcept(roomId string, exclude_ids []string, message Message) {
	b, err := json.Marshal(message)
	if err != nil {
		h.logger.Error(err)
	}
	h.room <- wsMessageForRoom{
		NodeId:     h.nodeId,
		RoomId:     roomId,
		Message:    b,
		ExcludeIds: exclude_ids,
	}
}

func (h *Hub[K, V]) BroadcastMsg(msg Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error(err)
	} else {
		h.broadcast <- b
	}
}

func (h *Hub[K, V]) doSendMsg(message wsDirectMessage[K, V]) {
	if h.clients[message.c] {
		select {
		case message.c.send <- message.message:
		default:
			delete(h.clients, message.c)
			go message.c.clean()
		}
	}
}

func (h *Hub[K, V]) doBroadcastMsg(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			delete(h.clients, client)
			go client.clean()
		}
	}
}

func (h *Hub[K, V]) doBroadcastRoomMsg(message wsMessageForRoom) {
	for client := range h.clients {
		if message.ExcludeIds != nil && containtString(message.ExcludeIds, client.ID) {
			continue
		}
		if client.exist(message.RoomId) {
			select {
			case client.send <- message.Message:
			default:
				delete(h.clients, client)
				go client.clean()
			}
		}
	}
}

func (h *Hub[K, V]) pushRoomMsgToRedis(message wsMessageForRoom) {
	b, _ := json.Marshal(message)
	if err := getRedis().Publish(context.Background(), h.pubSubRoomChannel, b).Err(); err != nil {
		h.logger.Error(err)
	}
}

func (h *Hub[K, V]) pushBroadcastMsgToRedis(message []byte) {
	msg := wsBroadcastMessage{
		NodeId:  h.nodeId,
		Message: message,
	}
	b, _ := json.Marshal(msg)
	getRedis().Publish(context.Background(), h.pubSubBroadcastChannel, b)
}

func (h *Hub[K, V]) processRedisRoomMsg(message *redis.Message) {
	m := wsMessageForRoom{}
	if err := json.Unmarshal([]byte(message.Payload), &m); err != nil {
		h.logger.Error(err)
	}
	if m.NodeId != h.nodeId {
		h.doBroadcastRoomMsg(m)
	}
}

func (h *Hub[K, V]) processRedisBroadcastMsg(message *redis.Message) {
	msg := wsBroadcastMessage{}
	if err := json.Unmarshal([]byte(message.Payload), &msg); err != nil {
		h.logger.Error(err)
	}
	if msg.NodeId != h.nodeId {
		h.doBroadcastMsg(msg.Message)
	}
}

func (h *Hub[K, V]) run() {
	for {
		select {
		// register and deregister client
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if h.clients[client] {
				delete(h.clients, client)
				go client.clean()
			}
		// send message to specific client
		case message := <-h.directMsg:
			h.doSendMsg(message)
		// broadcast and push message to redis channel
		case message := <-h.broadcast:
			go h.pushBroadcastMsgToRedis(message)
			h.doBroadcastMsg(message)
		// broadcast message for client in this node then push to redis channel
		case message := <-h.room:
			go h.pushRoomMsgToRedis(message)
			h.doBroadcastRoomMsg(message)
		// Two pubsub channel for receiving message from other node
		case message := <-h.subscribeRoomChan:
			h.processRedisRoomMsg(message)
		case message := <-h.subscribeBroadcastChan:
			h.processRedisBroadcastMsg(message)
		}
	}
}
