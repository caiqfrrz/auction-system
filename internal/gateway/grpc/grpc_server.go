package grpc

import (
	"auction-system/internal/gateway/sse"
	gatewayPb "auction-system/proto/gateway"
	lancePb "auction-system/proto/lance"
	pagamentoPb "auction-system/proto/pagamento"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

type GatewayGRPCServer struct {
	gatewayPb.UnimplementedGatewayServiceServer
	eventStream     *sse.EventStream
	lanceClient     lancePb.LanceServiceClient
	pagamentoClient pagamentoPb.PagamentoServiceClient
}

func NewGatewayGRPCServer(eventStream *sse.EventStream, lanceClient lancePb.LanceServiceClient, pagamentoClient pagamentoPb.PagamentoServiceClient) *GatewayGRPCServer {
	return &GatewayGRPCServer{
		eventStream:     eventStream,
		lanceClient:     lanceClient,
		pagamentoClient: pagamentoClient,
	}
}

func (s *GatewayGRPCServer) NotifyAuctionFinished(ctx context.Context, notif *gatewayPb.AuctionFinishedNotification) (*gatewayPb.Empty, error) {
	log.Printf("[Gateway gRPC] Auction finished notification: %s", notif.LeilaoId)

	// Query MSLance for winner
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.lanceClient.GetAuctionWinner(ctx, &lancePb.GetAuctionWinnerRequest{
		LeilaoId: notif.LeilaoId,
	})

	if err != nil {
		log.Printf("Error getting winner from MSLance: %v", err)
		return &gatewayPb.Empty{}, err
	}

	leilaoID, _ := strconv.Atoi(notif.LeilaoId)
	notification := sse.Notification{
		Type:     sse.LeilaoVencedor,
		LeilaoID: leilaoID,
		Data: map[string]interface{}{
			"vencedor_id": resp.UserId,
			"valor_final": resp.Valor,
			"leilao_id":   notif.LeilaoId,
		},
	}

	s.eventStream.Message <- notification
	log.Printf("SSE broadcast: Auction %s winner is %s (%.2f)", notif.LeilaoId, resp.UserId, resp.Valor)

	if resp.HasWinner {
		go func() {
			ctx := context.Background()
			_, err := s.pagamentoClient.NotifyAuctionWinner(ctx, &pagamentoPb.AuctionWinnerNotification{
				LeilaoId: notif.LeilaoId,
				UserId:   resp.UserId,
				Valor:    resp.Valor,
			})
			if err != nil {
				log.Printf("Error notifying MSPagamento: %v", err)
			} else {
				log.Printf("✅ MSPagamento notified about winner")
			}
		}()
	}

	return &gatewayPb.Empty{}, nil
}

func (s *GatewayGRPCServer) NotifyPaymentLink(ctx context.Context, notif *gatewayPb.PaymentLinkNotification) (*gatewayPb.Empty, error) {
	log.Printf("[Gateway gRPC] Payment link notification: user=%s, auction=%s", notif.UserId, notif.AuctionId)

	leilaoID, _ := strconv.Atoi(notif.AuctionId)
	notification := sse.Notification{
		Type:      sse.LinkPagamento,
		LeilaoID:  leilaoID,
		ClienteID: notif.UserId,
		Data: map[string]interface{}{
			"payment_link":   notif.PaymentLink,
			"transaction_id": notif.TransactionId,
			"auction_id":     notif.AuctionId,
		},
	}

	s.eventStream.Message <- notification
	log.Printf("SSE broadcast: Payment link sent to user %s", notif.UserId)

	return &gatewayPb.Empty{}, nil
}

func (s *GatewayGRPCServer) NotifyPaymentStatus(ctx context.Context, notif *gatewayPb.PaymentStatusNotification) (*gatewayPb.Empty, error) {
	log.Printf("[Gateway gRPC] Payment status notification: status=%s, auction=%s", notif.Status, notif.AuctionId)

	leilaoID, _ := strconv.Atoi(notif.AuctionId)
	notification := sse.Notification{
		Type:      sse.StatusPagamento,
		LeilaoID:  leilaoID,
		ClienteID: notif.WinnerId,
		Data: map[string]interface{}{
			"status":         notif.Status,
			"transaction_id": notif.TransactionId,
			"auction_id":     notif.AuctionId,
			"amount":         notif.Amount,
		},
	}

	s.eventStream.Message <- notification
	log.Printf("SSE broadcast: Payment status %s sent to user %s", notif.Status, notif.WinnerId)

	return &gatewayPb.Empty{}, nil
}

func StartGRPCServer(eventStream *sse.EventStream, lanceClient lancePb.LanceServiceClient, pagamentoClient pagamentoPb.PagamentoServiceClient, port string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	gatewayPb.RegisterGatewayServiceServer(grpcServer, NewGatewayGRPCServer(eventStream, lanceClient, pagamentoClient))

	log.Printf("🚀 Gateway gRPC server listening on port %s", port)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve Gateway gRPC: %v", err)
		}
	}()

	return grpcServer, nil
}
