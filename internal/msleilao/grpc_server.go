package msleilao

import (
	pb "auction-system/proto/leilao"
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
)

type LeilaoGRPCServer struct {
	pb.UnimplementedLeilaoServiceServer
	msLeilao *MsLeilao
}

func NewLeilaoGRPCServer(ms *MsLeilao) *LeilaoGRPCServer {
	return &LeilaoGRPCServer{msLeilao: ms}
}

func (s *LeilaoGRPCServer) CreateAuction(ctx context.Context, req *pb.CreateAuctionRequest) (*pb.CreateAuctionResponse, error) {
	log.Printf("[MSLeilao gRPC] CreateAuction: %s", req.Titulo)

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(req.DuracaoSegundos) * time.Second)

	err := s.msLeilao.CreateAuction(req.Titulo, startTime, endTime)

	if err != nil {
		return &pb.CreateAuctionResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	auctions := s.msLeilao.ConsultAuctions()
	lastAuction := auctions[len(auctions)-1]

	return &pb.CreateAuctionResponse{
		LeilaoId: lastAuction.ID,
		Success:  true,
	}, nil
}

func (s *LeilaoGRPCServer) ConsultAuctions(ctx context.Context, req *pb.ConsultAuctionsRequest) (*pb.ConsultAuctionsResponse, error) {
	log.Printf("[MSLeilao gRPC] ConsultAuctions")

	auctions := s.msLeilao.ConsultAuctions()

	var pbAuctions []*pb.Auction
	for _, a := range auctions {
		tempoRestante := int64(0)
		if a.Ativo {
			tempoRestante = int64(time.Until(a.Fim).Seconds())
			if tempoRestante < 0 {
				tempoRestante = 0
			}
		}

		pbAuctions = append(pbAuctions, &pb.Auction{
			Id:            a.ID,
			Titulo:        a.Descricao,
			Descricao:     a.Descricao,
			ValorInicial:  0,
			TempoRestante: tempoRestante,
			Active:        a.Ativo,
		})
	}

	return &pb.ConsultAuctionsResponse{Auctions: pbAuctions}, nil
}

func (s *LeilaoGRPCServer) NotifyAuctionStarted(ctx context.Context, notif *pb.AuctionStartedNotification) (*pb.Empty, error) {
	log.Printf("[MSLeilao gRPC] Auction started: %s", notif.LeilaoId)
	return &pb.Empty{}, nil
}

func (s *LeilaoGRPCServer) NotifyAuctionFinished(ctx context.Context, notif *pb.AuctionFinishedNotification) (*pb.Empty, error) {
	log.Printf("[MSLeilao gRPC] Auction finished: %s", notif.LeilaoId)
	return &pb.Empty{}, nil
}

func StartGRPCServer(msLeilao *MsLeilao, port string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLeilaoServiceServer(grpcServer, NewLeilaoGRPCServer(msLeilao))

	log.Printf("🚀 MSLeilao gRPC server listening on port %s", port)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	return grpcServer, nil
}
