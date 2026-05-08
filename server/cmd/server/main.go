package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	gamehandler "ssg/handler"
)

func main() {
	port := flag.String("port", "8080", "server port")
	flag.Parse()

	s, err := gamehandler.NewServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create server: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Starting server on port %s", *port)
	if err := s.Start(*port); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
