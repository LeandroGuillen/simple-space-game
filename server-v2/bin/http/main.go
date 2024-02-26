package main

import (
	"lejn/ssg/bin/http/server"
)

func main() {
	s := server.NewServer()
	s.Start()
}
