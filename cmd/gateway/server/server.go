package server

import (
	grpcClients "auction-system/internal/gateway/grpc"
	"auction-system/internal/gateway/sse"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	port        string
	grpcClients *grpcClients.GRPCClients
	eventStream *sse.EventStream
}

func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

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
		return nil, fmt.Errorf("failed to create gRPC clients: %w", err)
	}

	newStream := sse.NewEventStream()
	//go newStream.Listen()

	newServer := &Server{
		port:        strconv.Itoa(port),
		grpcClients: grpcCli,
		eventStream: newStream,
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      newServer.registerRoutes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Gateway server initialized on port %s", port)
	return server, nil
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
