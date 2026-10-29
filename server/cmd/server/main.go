package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	gamehandler "ssg/handler"
	"ssg/version"
)

func main() {
	port := flag.String("port", "8080", "server port")
	debugFlag := flag.Bool("debug", false, "enable /debug/* endpoints (also SSG_DEBUG=1)")
	flag.Parse()

	debug := *debugFlag || envTruthy("SSG_DEBUG")

	s, err := gamehandler.NewServerWithOptions(gamehandler.ServerOptions{Debug: debug})
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create server: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Simple Space Game server version %s", version.Version())
	if debug {
		log.Printf("Debug endpoints enabled")
	}
	log.Printf("Starting server on port %s", *port)
	if err := s.Start(*port); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}

func envTruthy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
