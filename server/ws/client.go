package ws

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub      *Hub
	socket   *websocket.Conn
	send     chan gin.H
	gameID   string
	playerID uint
}

func (c *Client) read() {
	defer func() {
		c.hub.unregister <- c
		c.socket.Close()
	}()
	for {
		var data map[string]interface{}
		err := c.socket.ReadJSON(&data)
		if err != nil {
			log.Println("ws read error:", err)
			break
		}
		c.hub.broadcast <- data
	}
}

func (c *Client) write() {
	defer c.socket.Close()
	for {
		message, ok := <-c.send

		if !ok {
			// The hub closed the channel.
			_ = c.socket.WriteJSON(gin.H{})
			return
		}

		err := c.socket.WriteJSON(message)
		if err != nil {
			log.Println("ws write error:", err)
			break
		}
	}
}

// Handler handles websocket requests from the peer.
func Handler(hub *Hub, c *gin.Context) {
	gameID := c.Query("game_id")
	if gameID == "" {
		log.Println("ws missing game_id query parameter")
		return
	}
	playerIDRaw := c.Query("player_id")
	if playerIDRaw == "" {
		log.Println("ws missing player_id query parameter")
		return
	}
	playerID64, err := strconv.ParseUint(playerIDRaw, 10, 64)
	if err != nil {
		log.Println("ws invalid player_id:", err)
		return
	}

	socket, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		hub:      hub,
		socket:   socket,
		send:     make(chan gin.H, 1),
		gameID:   gameID,
		playerID: uint(playerID64),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.write()
	go client.read()
}
