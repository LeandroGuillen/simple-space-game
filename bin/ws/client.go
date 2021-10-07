// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub
	// The websocket connection.
	socket *websocket.Conn
	// Buffered channel of outbound messages.
	send chan gin.H
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

// wsHandler handles websocket requests from the peer.
func wsHandler(hub *Hub, c *gin.Context) {
	socket, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, socket: socket, send: make(chan gin.H)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.write()
	go client.read()
}
