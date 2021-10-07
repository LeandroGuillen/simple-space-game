// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"
)

var addr = flag.String("addr", ":8088", "http service address")

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		wsHandler(hub, c)
	})
	err := r.Run(*addr)
	if err != nil {
		log.Printf("API error: %s", err)
	}
}
