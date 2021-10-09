package main

import (
	"math/rand"
	"ssg/bin/http-server/server"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	srv := server.NewServer()
	srv.Start()
}
