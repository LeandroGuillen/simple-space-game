package main

import (
	"corion/bin/http-server/server"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	srv := server.NewServer()
	srv.Start()
}
