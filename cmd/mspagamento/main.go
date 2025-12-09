package main

import (
	"auction-system/internal/mspagamento"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("cmd/mspagamento/.env")

	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:8083"
	}

	externalPayURL := os.Getenv("PAGEXTERNO_URL")
	if externalPayURL == "" {
		externalPayURL = "http://localhost:8085"
	}

	msPagamento := mspagamento.NewMsPagamento(publicURL, externalPayURL)

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50053"
	}

	grpcServer, err := mspagamento.StartGRPCServer(msPagamento, grpcPort)
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

	go func() {
		if err := msPagamento.StartWebhookServer(":8083"); err != nil {
			log.Fatalf("Failed to start webhook server: %v", err)
		}
	}()

	log.Println("MSPagamento iniciado")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down MSPagamento...")
	grpcServer.GracefulStop()
}
