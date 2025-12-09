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
	msLance       *MSLance
	notifyGateway func(eventType string, data interface{})
}

func NewMsLanceGRPCServer(ms *MSLance) *MsLanceGRPCServer {
	server := &MsLanceGRPCServer{msLance: ms}

	ms.SetBidCallbacks(
		server.onBidValidated,
		server.onBidInvalidated,
		server.onAuctionWinner,
	)

	return server
}

func (s *MsLanceGRPCServer) onBidValidated(bid models.LanceRealizado) {
	log.Printf("[Callback] Lance validado: %+v", bid)
	if s.notifyGateway != nil {
		s.notifyGateway("lance_validado", bid)
	}
}

func (s *MsLanceGRPCServer) onBidInvalidated(leilaoID, userID string, valor float64, motivo string) {
	log.Printf("[Callback] Lance invalidado: leilao=%s, user=%s, motivo=%s", leilaoID, userID, motivo)

	data := map[string]interface{}{
		"leilao_id": leilaoID,
		"user_id":   userID,
		"valor":     valor,
		"motivo":    motivo,
	}

	if s.notifyGateway != nil {
		s.notifyGateway("lance_invalidado", data)
	}
}

func (s *MsLanceGRPCServer) onAuctionWinner(vencedor models.LeilaoVencedor) {
	log.Printf("[Callback] Vencedor: %+v", vencedor)
	if s.notifyGateway != nil {
		s.notifyGateway("leilao_vencedor", vencedor)
	}
}

func (s *MsLanceGRPCServer) SetGatewayCallback(callback func(string, interface{})) {
	s.notifyGateway = callback
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

func (s *MsLanceGRPCServer) NotifyBidValidated(ctx context.Context, notif *pb.BidValidatedNotification) (*pb.Empty, error) {
	log.Printf("[MSLance gRPC] Bid validated: %s - %.2f", notif.LeilaoId, notif.Valor)
	return &pb.Empty{}, nil
}

func (s *MsLanceGRPCServer) NotifyBidInvalidated(ctx context.Context, notif *pb.BidInvalidatedNotification) (*pb.Empty, error) {
	log.Printf("[MSLance gRPC] Bid invalidated: %s - %s", notif.LeilaoId, notif.Motivo)
	return &pb.Empty{}, nil
}

func StartGRPCServer(msLance *MSLance, port string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLanceServiceServer(grpcServer, NewMsLanceGRPCServer(msLance))

	log.Printf("🚀 MSLance gRPC server listening on port %s", port)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return grpcServer, nil
}
