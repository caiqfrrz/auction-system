package main

import (
	"auction-system/internal/msleilao"
	gatewayPb "auction-system/proto/gateway"
	lancePb "auction-system/proto/lance"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	godotenv.Load("cmd/msleilao/.env")

	msLeilao := msleilao.NewMsLeilao()

	// Connect to MSLance
	lanceAddr := os.Getenv("MSLANCE_GRPC")
	if lanceAddr == "" {
		lanceAddr = "localhost:50052"
	}

	lanceConn, err := grpc.Dial(lanceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to MSLance: %v", err)
	}
	defer lanceConn.Close()

	lanceClient := lancePb.NewLanceServiceClient(lanceConn)

	// Connect to Gateway
	gatewayAddr := os.Getenv("GATEWAY_GRPC")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:50060"
	}

	gatewayConn, err := grpc.Dial(gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Gateway: %v", err)
	}
	defer gatewayConn.Close()

	gatewayClient := gatewayPb.NewGatewayServiceClient(gatewayConn)

	// Set callbacks
	msLeilao.SetAuctionCallbacks(
		// onAuctionStarted
		func(leilaoID string, duracao int64) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := lanceClient.NotifyAuctionStarted(ctx, &lancePb.AuctionStartedNotification{
				LeilaoId: leilaoID,
				Duracao:  duracao,
			})
			if err != nil {
				log.Printf("Error notifying MSLance: %v", err)
			} else {
				log.Printf("✅ MSLance notified: auction %s started", leilaoID)
			}
		},
		// onAuctionFinished
		func(leilaoID string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Notify MSLance
			_, err := lanceClient.NotifyAuctionFinished(ctx, &lancePb.AuctionFinishedNotification{
				LeilaoId: leilaoID,
			})
			if err != nil {
				log.Printf("Error notifying MSLance: %v", err)
			} else {
				log.Printf("✅ MSLance notified: auction %s finished", leilaoID)
			}

			// Notify Gateway
			_, err = gatewayClient.NotifyAuctionFinished(ctx, &gatewayPb.AuctionFinishedNotification{
				LeilaoId: leilaoID,
			})
			if err != nil {
				log.Printf("Error notifying Gateway: %v", err)
			} else {
				log.Printf("✅ Gateway notified: auction %s finished", leilaoID)
			}
		},
	)

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

	log.Println("Shutting down MSLeilao...")
	grpcServer.GracefulStop()
}
