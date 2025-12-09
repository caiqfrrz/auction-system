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
	log.Printf("[MSLeilao gRPC] CreateAuction: %s", req.Description)

	startTime, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		return &pb.CreateAuctionResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid start time: %v", err),
		}, nil
	}

	endTime, err := time.Parse(time.RFC3339, req.End)
	if err != nil {
		return &pb.CreateAuctionResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid end time: %v", err),
		}, nil
	}

	err = s.msLeilao.CreateAuction(req.Description, startTime, endTime)

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
		pbAuctions = append(pbAuctions, &pb.Auction{
			Id:          a.ID,
			Description: a.Descricao,
			Start:       a.Inicio.Format(time.RFC3339),
			End:         a.Fim.Format(time.RFC3339),
			Active:      a.Ativo,
		})
	}

	return &pb.ConsultAuctionsResponse{Auctions: pbAuctions}, nil
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
