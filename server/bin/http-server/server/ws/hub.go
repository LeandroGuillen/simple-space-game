// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ws

import (
	"ssg/game"

	"github.com/gin-gonic/gin"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan gin.H

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	gameUpdates chan *game.Game

	server ServerGetter
}

type ServerGetter interface {
	GetGame(id string) *game.Game
}

func NewHub(server ServerGetter) *Hub {
	return &Hub{
		broadcast:   make(chan gin.H, 10),
		register:    make(chan *Client, 10),
		unregister:  make(chan *Client, 10),
		clients:     make(map[*Client]bool),
		gameUpdates: make(chan *game.Game, 10),
		server:      server,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			if g := h.server.GetGame(client.gameID); g != nil {
				visible, err := g.GetVisibleGameForPlayer(client.playerID)
				if err == nil {
					select {
					case client.send <- gin.H{"game_update": visible}:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case g := <-h.gameUpdates:
			for client := range h.clients {
				if client.gameID != g.ID {
					continue
				}
				visible, err := g.GetVisibleGameForPlayer(client.playerID)
				if err != nil {
					continue
				}
				select {
				case client.send <- gin.H{"game_update": visible}:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) BroadcastGameUpdate(g *game.Game) {
	h.gameUpdates <- g
}
