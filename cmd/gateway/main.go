package main

import (
	"auction-system/cmd/gateway/server"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("cmd/gateway/.env"); err != nil {
		fmt.Println("Warning: .env file not found, using defaults")
	}

	httpServer, grpcServer, err := server.NewServer()
	if err != nil {
		fmt.Printf("error starting server: %s", err.Error())
		return
	}

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err = httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %s", err)
		}
	}()

	<-quit
	log.Println("Shutting down Gateway...")
	grpcServer.GracefulStop()
}
