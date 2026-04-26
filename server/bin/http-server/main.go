package main

import (
	"ssg/bin/http-server/server"
)

func main() {
	newServer := server.NewServer()
	newServer.Start()
}
