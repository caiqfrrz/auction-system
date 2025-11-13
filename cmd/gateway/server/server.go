package server

import (
	"auction-system/internal/gateway/rabbitmq"
	"auction-system/internal/gateway/sse"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	port           int
	msLanceHost    string
	msLeilaoHost   string
	eventStream    *sse.EventStream
	rabbitConsumer *rabbitmq.RabbitMQConsumer
}

func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	msLeilao := os.Getenv("MSLEILAO_HOST")
	msLance := os.Getenv("MSLANCE_HOST")
	rabbitURL := os.Getenv("RABBITMQ_URL")

	newStream := sse.NewEventStream()

	rabbitConsumer, err := rabbitmq.NewRabbitMQConsumer(rabbitURL, newStream)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ consumer: %w", err)
	}

	if err := rabbitConsumer.ConsumeQueues(); err != nil {
		rabbitConsumer.Close()
		return nil, fmt.Errorf("failed to start consuming queues: %w", err)
	}

	NewServer := &Server{
		port:           port,
		msLanceHost:    msLance,
		msLeilaoHost:   msLeilao,
		eventStream:    newStream,
		rabbitConsumer: rabbitConsumer,
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.registerRoutes(),
		IdleTimeout:  0,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
	}

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

	r.GET("/consult-auctions", s.ConsultAuctions)
	r.GET("/register-interest/:auctionID/stream", HeadersMiddleware(), s.eventStream.SSEConnMiddleware(), s.RegisterInterest)
	r.GET("/cancel-interest", s.CancelInterest)
	r.GET("/highest-bid", s.GetHighestBid)
	r.POST("/create-auction", s.CreateAuction)
	r.POST("/make-bid", s.PlaceBid)

	return r
}
