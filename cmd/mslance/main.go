package main

import (
	"auction-system/internal/mslance"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("cmd/mslance/.env")

	msLance := mslance.NewMSLance()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50052"
	}

	grpcServer, err := mslance.StartGRPCServer(msLance, grpcPort)
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

	log.Println("MSLance iniciado")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down MSLance...")
	grpcServer.GracefulStop()
}
