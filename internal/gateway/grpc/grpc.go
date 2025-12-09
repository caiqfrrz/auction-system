package grpc

import (
	lancePb "auction-system/proto/lance"
	leilaoPb "auction-system/proto/leilao"
	pagamentoPb "auction-system/proto/pagamento"
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClients struct {
	LeilaoClient    leilaoPb.LeilaoServiceClient
	LanceClient     lancePb.LanceServiceClient
	PagamentoClient pagamentoPb.PagamentoServiceClient

	leilaoConn    *grpc.ClientConn
	lanceConn     *grpc.ClientConn
	pagamentoConn *grpc.ClientConn
}

func NewGRPCClients(leilaoAddr, lanceAddr, pagamentoAddr string) (*GRPCClients, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("Connecting to MSLeilao at %s...", leilaoAddr)
	leilaoConn, err := grpc.DialContext(ctx, leilaoAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MSLeilao: %w", err)
	}

	log.Printf("Connecting to MSLance at %s...", lanceAddr)
	lanceConn, err := grpc.DialContext(ctx, lanceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		leilaoConn.Close()
		return nil, fmt.Errorf("failed to connect to MSLance: %w", err)
	}

	log.Printf("Connecting to MSPagamento at %s...", pagamentoAddr)
	pagamentoConn, err := grpc.DialContext(ctx, pagamentoAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		leilaoConn.Close()
		lanceConn.Close()
		return nil, fmt.Errorf("failed to connect to MSPagamento: %w", err)
	}

	log.Println("All gRPC clients connected successfully")

	return &GRPCClients{
		LeilaoClient:    leilaoPb.NewLeilaoServiceClient(leilaoConn),
		LanceClient:     lancePb.NewLanceServiceClient(lanceConn),
		PagamentoClient: pagamentoPb.NewPagamentoServiceClient(pagamentoConn),
		leilaoConn:      leilaoConn,
		lanceConn:       lanceConn,
		pagamentoConn:   pagamentoConn,
	}, nil
}

func (c *GRPCClients) Close() {
	if c.leilaoConn != nil {
		c.leilaoConn.Close()
	}
	if c.lanceConn != nil {
		c.lanceConn.Close()
	}
	if c.pagamentoConn != nil {
		c.pagamentoConn.Close()
	}
}
