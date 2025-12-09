package main

import (
	"auction-system/internal/mspagamento"
	"auction-system/pkg/models"
	gatewayPb "auction-system/proto/gateway"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	gatewayAddr := os.Getenv("GATEWAY_GRPC")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:50060"
	}

	gatewayConn, err := grpc.Dial(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Warning: Could not connect to Gateway: %v", err)
	}
	defer gatewayConn.Close()

	gatewayClient := gatewayPb.NewGatewayServiceClient(gatewayConn)
	msPagamento := mspagamento.NewMsPagamento(publicURL, externalPayURL)

	msPagamento.SetPaymentCallbacks(
		// onPaymentLink
		func(linkPag models.LinkPagamento) {
			ctx := context.Background()
			_, err := gatewayClient.NotifyPaymentLink(ctx, &gatewayPb.PaymentLinkNotification{
				UserId:        linkPag.UserID,
				PaymentLink:   linkPag.PaymentLink,
				TransactionId: linkPag.TransactionID,
				AuctionId:     linkPag.AuctionID,
			})
			if err != nil {
				log.Printf("Error notifying Gateway about payment link: %v", err)
			} else {
				log.Printf("✅ Gateway notified: payment link sent")
			}
		},
		// onPaymentStatus
		func(status models.StatusPagamento) {
			ctx := context.Background()
			_, err := gatewayClient.NotifyPaymentStatus(ctx, &gatewayPb.PaymentStatusNotification{
				TransactionId: status.TransactionID,
				Status:        status.Status,
				AuctionId:     status.AuctionID,
				WinnerId:      status.WinnerID,
				Amount:        status.Amount,
			})
			if err != nil {
				log.Printf("Error notifying Gateway about payment status: %v", err)
			} else {
				log.Printf("✅ Gateway notified: payment status %s", status.Status)
			}
		},
	)

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
