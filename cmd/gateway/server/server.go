package server

import (
	gatewayGrpc "auction-system/internal/gateway/grpc"
	grpcClients "auction-system/internal/gateway/grpc"
	"auction-system/internal/gateway/sse"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type Server struct {
	port            string
	grpcClients     *grpcClients.GRPCClients
	eventStream     *sse.EventStream
	gatewayGrpcPort string
}

func NewServer() (*http.Server, *grpc.Server, error) {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080"
	}

	leilaoAddr := os.Getenv("MSLEILAO_GRPC")
	if leilaoAddr == "" {
		leilaoAddr = "localhost:50051"
	}

	lanceAddr := os.Getenv("MSLANCE_GRPC")
	if lanceAddr == "" {
		lanceAddr = "localhost:50052"
	}

	pagamentoAddr := os.Getenv("MSPAGAMENTO_GRPC")
	if pagamentoAddr == "" {
		pagamentoAddr = "localhost:50053"
	}

	grpcCli, err := grpcClients.NewGRPCClients(leilaoAddr, lanceAddr, pagamentoAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gRPC clients: %w", err)
	}

	newStream := sse.NewEventStream()

	// Start Gateway gRPC server
	gatewayGrpcPort := os.Getenv("GATEWAY_GRPC_PORT")
	if gatewayGrpcPort == "" {
		gatewayGrpcPort = "50060"
	}

	gatewayGrpcServer, err := gatewayGrpc.StartGRPCServer(newStream, grpcCli.LanceClient, grpcCli.PagamentoClient, gatewayGrpcPort)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start Gateway gRPC server: %w", err)
	}

	newServer := &Server{
		port:            portStr,
		grpcClients:     grpcCli,
		eventStream:     newStream,
		gatewayGrpcPort: gatewayGrpcPort,
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", portStr),
		Handler: newServer.registerRoutes(),
	}

	log.Printf("Gateway server initialized on port %s", portStr)
	return server, gatewayGrpcServer, nil
}

func (s *Server) registerRoutes() http.Handler {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// SSE
	r.GET("/consult-auctions", s.ConsultAuctions)
	r.GET("/register-interest/:auctionID/stream", HeadersMiddleware(), s.eventStream.SSEConnMiddleware(), s.RegisterInterest)

	// REST endpoints (frontend -> gateway)
	r.GET("/cancel-interest", s.CancelInterest)
	r.GET("/highest-bid", s.GetHighestBid)
	r.POST("/create-auction", s.CreateAuction)
	r.POST("/make-bid", s.PlaceBid)

	return r
}
