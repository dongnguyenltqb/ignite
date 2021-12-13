// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
func (h *Hub) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{
		Id:                 uuid.New().String(),
		hub:                h,
		conn:               conn,
		send:               make(chan []byte, 256),
		rchan:              make(chan wsRoomActionMessage, 100),
		rooms:              make([]string, maxRoomSize),
		onCloseHandelFuncs: []func(string){},
		logger:             getLogger(),
	}
	client.rooms = []string{client.Id}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.roomPump()
	go client.writePump()
	go client.readPump()
	h.OnNewClient(client)

}

// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Namespace
	namespace string
	// Node Id
	nodeId string
	// Registered clients.
	clients map[*Client]bool

	// Outbound message for specific client
	directMsg chan wsDirectMessage

	// Outbound messages
	broadcast chan []byte

	// Outbound message for a specific room
	room chan wsMessageForRoom

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Pubsub channel
	pubSubRoomChannel      string
	pubSubBroadcastChannel string
	subscribeRoomChan      <-chan *redis.Message
	subscribeBroadcastChan <-chan *redis.Message

	// Logger
	logger *logrus.Logger

	// Emit function for register handle func
	OnNewClient func(*Client)
}

func newHub(namespace string) *Hub {
	redisSubscribeRoom := getRedis().Subscribe(context.Background(), pubSubRoomChannel+namespace)
	redisSubscribeBroadcast := getRedis().Subscribe(context.Background(), pubSubBroadcastChannel+namespace)
	hub := &Hub{
		namespace:              namespace,
		nodeId:                 uuid.New().String(),
		directMsg:              make(chan wsDirectMessage),
		broadcast:              make(chan []byte),
		room:                   make(chan wsMessageForRoom),
		register:               make(chan *Client),
		unregister:             make(chan *Client),
		clients:                make(map[*Client]bool),
		pubSubRoomChannel:      pubSubRoomChannel,
		pubSubBroadcastChannel: pubSubBroadcastChannel,
		subscribeRoomChan:      redisSubscribeRoom.Channel(),
		subscribeBroadcastChan: redisSubscribeBroadcast.Channel(),
		logger:                 getLogger(),
	}
	go hub.run()
	return hub
}

// Send message to specific room
func (h *Hub) SendMsgToRoom(roomId string, message Message) {
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
func (h *Hub) SendMsgToRoomWithExcludeClient(roomId string, exclude_ids []string, message Message) {
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

func (h *Hub) BroadcastMsg(msg Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error(err)
	} else {
		h.broadcast <- b
	}
}

func (h *Hub) doSendMsg(message wsDirectMessage) {
	if ok := h.clients[message.c]; ok {
		select {
		case message.c.send <- message.message:
		default:
			delete(h.clients, message.c)
			go message.c.clean()
		}
	}
}

func (h *Hub) doBroadcastMsg(message []byte) {
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			delete(h.clients, client)
			go client.clean()
		}
	}
}

func (h *Hub) doBroadcastRoomMsg(message wsMessageForRoom) {
	if message.ExcludeIds != nil {
		for client := range h.clients {
			ok := client.exist(message.RoomId)
			if ok {
				select {
				case client.send <- message.Message:
				default:
					delete(h.clients, client)
					go client.clean()
				}
			}
		}
	} else {
		for client := range h.clients {
			exclude := false
			for _, excludeId := range message.ExcludeIds {
				if excludeId == client.Id {
					exclude = true
				}
			}
			if exclude {
				continue
			}
			ok := client.exist(message.RoomId)
			if ok {
				select {
				case client.send <- message.Message:
				default:
					delete(h.clients, client)
					go client.clean()
				}
			}
		}
	}
}

func (h *Hub) pushRoomMsgToRedis(message wsMessageForRoom) {
	b, _ := json.Marshal(message)
	if err := getRedis().Publish(context.Background(), h.pubSubRoomChannel, b).Err(); err != nil {
		h.logger.Error(err)
	}
}

func (h *Hub) pushBroadcastMsgToRedis(message []byte) {
	msg := wsBroadcastMessage{
		NodeId:  h.nodeId,
		Message: message,
	}
	b, _ := json.Marshal(msg)
	getRedis().Publish(context.Background(), h.pubSubBroadcastChannel, b)
}

func (h *Hub) processRedisRoomMsg(message *redis.Message) {
	m := wsMessageForRoom{}
	if err := json.Unmarshal([]byte(message.Payload), &m); err != nil {
		h.logger.Error(err)
	}
	if m.NodeId != h.nodeId {
		h.doBroadcastRoomMsg(m)
	}
}

func (h *Hub) processRedisBroadcastMsg(message *redis.Message) {
	msg := wsBroadcastMessage{}
	if err := json.Unmarshal([]byte(message.Payload), &msg); err != nil {
		h.logger.Error(err)
	}
	if msg.NodeId != h.nodeId {
		h.doBroadcastMsg(msg.Message)
	}
}

func (h *Hub) run() {
	for {
		select {
		// register and deregister client
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
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
