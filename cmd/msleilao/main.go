package main

import (
	"auction-system/internal/msleilao"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("cmd/msleilao/.env")

	msLeilao := msleilao.NewMsLeilao()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	grpcServer, err := msleilao.StartGRPCServer(msLeilao, grpcPort)
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

	log.Println("✅ MSLeilao iniciado")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down MSLeilao...")
	grpcServer.GracefulStop()
}
