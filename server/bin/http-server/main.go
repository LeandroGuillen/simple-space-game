package main

import (
	"math/rand"
	"ssg/bin/http-server/server"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	newServer := server.NewServer()
	newServer.Start()
}
