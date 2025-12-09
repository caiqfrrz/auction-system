package mslance

import (
	"auction-system/pkg/models"
	pb "auction-system/proto/lance"
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

type MsLanceGRPCServer struct {
	pb.UnimplementedLanceServiceServer
	msLance *MSLance
}

func NewMsLanceGRPCServer(ms *MSLance) *MsLanceGRPCServer {
	return &MsLanceGRPCServer{msLance: ms}
}

func (s *MsLanceGRPCServer) MakeBid(ctx context.Context, req *pb.MakeBidRequest) (*pb.MakeBidResponse, error) {
	log.Printf("[MSLance gRPC] MakeBid: user=%s, leilao=%s, valor=%.2f",
		req.UserId, req.LeilaoId, req.Valor)

	bid := models.LanceRealizado{
		UserID:   req.UserId,
		LeilaoID: req.LeilaoId,
		Valor:    req.Valor,
	}

	err := s.msLance.MakeBid(bid)
	if err != nil {
		return &pb.MakeBidResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.MakeBidResponse{Success: true}, nil
}

func (s *MsLanceGRPCServer) GetHighestBid(ctx context.Context, req *pb.GetHighestBidRequest) (*pb.GetHighestBidResponse, error) {
	log.Printf("[MSLance gRPC] GetHighestBid: leilao=%s", req.LeilaoId)

	highestBid, err := s.msLance.GetHighestBid(req.LeilaoId)
	if err != nil {
		return nil, err
	}

	return &pb.GetHighestBidResponse{
		HighestBid: highestBid,
		LeilaoId:   req.LeilaoId,
	}, nil
}

func (s *MsLanceGRPCServer) GetAuctionWinner(ctx context.Context, req *pb.GetAuctionWinnerRequest) (*pb.GetAuctionWinnerResponse, error) {
	log.Printf("[MSLance gRPC] GetAuctionWinner: leilao=%s", req.LeilaoId)

	userID, valor, hasWinner := s.msLance.GetAuctionWinner(req.LeilaoId)

	return &pb.GetAuctionWinnerResponse{
		UserId:    userID,
		Valor:     valor,
		HasWinner: hasWinner,
	}, nil
}

func (s *MsLanceGRPCServer) NotifyAuctionStarted(ctx context.Context, notif *pb.AuctionStartedNotification) (*pb.Empty, error) {
	log.Printf("[MSLance gRPC] Auction started: %s", notif.LeilaoId)

	s.msLance.mu.Lock()
	s.msLance.leiloes[notif.LeilaoId] = &LeilaoStatus{
		ID:         notif.LeilaoId,
		Descricao:  "",
		Ativo:      true,
		MaiorLance: 0,
		Vencedor:   "",
	}
	s.msLance.mu.Unlock()

	log.Printf("Leilão %s registrado no MSLance", notif.LeilaoId)
	return &pb.Empty{}, nil
}

func (s *MsLanceGRPCServer) NotifyAuctionFinished(ctx context.Context, notif *pb.AuctionFinishedNotification) (*pb.Empty, error) {
	log.Printf("[MSLance gRPC] Auction finished: %s", notif.LeilaoId)

	s.msLance.mu.Lock()
	defer s.msLance.mu.Unlock()

	leilao, ok := s.msLance.leiloes[notif.LeilaoId]
	if ok && leilao.Ativo {
		leilao.Ativo = false
		log.Printf("Leilão %s finalizado. Vencedor: %s (%.2f)",
			leilao.ID, leilao.Vencedor, leilao.MaiorLance)
	}

	return &pb.Empty{}, nil
}

func StartGRPCServer(serverInstance *MsLanceGRPCServer, port string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLanceServiceServer(grpcServer, serverInstance)

	log.Printf("🚀 MSLance gRPC server listening on port %s", port)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return grpcServer, nil
}
