// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ws

import (
	"ssg/game"

	"github.com/gin-gonic/gin"
)

type Hub struct {
	clients     map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	gameUpdates chan *game.Game
	server      ServerGetter
}

type ServerGetter interface {
	GetGame(id string) *game.Game
}

func NewHub(s ServerGetter) *Hub {
	return &Hub{
		register:    make(chan *Client, 10),
		unregister:  make(chan *Client, 10),
		clients:     make(map[*Client]bool),
		gameUpdates: make(chan *game.Game, 10),
		server:      s,
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
		case g := <-h.gameUpdates:
			for client := range h.clients {
				// ID is immutable after NewGame; GetVisibleGameForPlayer takes RLock.
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
					// Slow / stuck client: drop rather than stall the hub.
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
