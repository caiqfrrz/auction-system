package mspagamento

import (
	"auction-system/pkg/models"
	pb "auction-system/proto/pagamento"
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

type PagamentoGRPCServer struct {
	pb.UnimplementedPagamentoServiceServer
	msPagamento *MsPagamento
}

func NewPagamentoGRPCServer(ms *MsPagamento) *PagamentoGRPCServer {
	return &PagamentoGRPCServer{msPagamento: ms}
}

func (s *PagamentoGRPCServer) NotifyAuctionWinner(ctx context.Context, notif *pb.AuctionWinnerNotification) (*pb.Empty, error) {
	log.Printf("[MSPagamento gRPC] Auction winner: leilao=%s, user=%s, valor=%.2f",
		notif.LeilaoId, notif.UserId, notif.Valor)

	leilaoVencedor := models.LeilaoVencedor{
		LeilaoID: notif.LeilaoId,
		UserID:   notif.UserId,
		Valor:    notif.Valor,
	}

	if err := s.msPagamento.SubmitPaymentData(leilaoVencedor); err != nil {
		log.Printf("Error processing payment: %v", err)
		return nil, err
	}

	return &pb.Empty{}, nil
}

func StartGRPCServer(msPagamento *MsPagamento, port string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPagamentoServiceServer(grpcServer, NewPagamentoGRPCServer(msPagamento))

	log.Printf("MSPagamento gRPC server listening on port %s", port)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return grpcServer, nil
}
