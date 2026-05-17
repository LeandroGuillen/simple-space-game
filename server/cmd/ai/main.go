package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ssg/ai"
	"ssg/ai/ui"

	"github.com/gin-gonic/gin"
)

func main() {
	listen := flag.String("listen", ":8090", "address to listen on for the AI manager UI")
	defaultServer := flag.String("default-server", "http://127.0.0.1:8080", "default game server URL prefilled in the UI")
	tickDelay := flag.Duration("tick-delay", ai.DefaultTickDelay, "minimum delay between an agent detecting its turn and submitting; keeps fast-advancing games inspectable")
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	manager := ai.NewManager(rootCtx, *tickDelay)
	ui.Mount(router, manager, *defaultServer)

	srv := &http.Server{
		Addr:    *listen,
		Handler: router,
	}

	go func() {
		log.Printf("AI manager listening on %s", *listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
			rootCancel()
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs
	log.Printf("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	manager.Shutdown()
}
