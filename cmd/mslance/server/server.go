package server

import (
	"auction-system/internal/mslance"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Server struct {
	msLance *mslance.MSLance
}

func NewServer(ch *amqp.Channel) *http.Server {
	msLance := mslance.NewMSLance(ch)

	NewServer := &Server{
		msLance: msLance,
	}

	server := &http.Server{
		Addr:         ":8082",
		Handler:      NewServer.registerRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	msLance.DeclareExchangeAndQueues()
	msLance.ListenLeilaoIniciado()
	msLance.ListenLeilaoFinalizado()

	return server
}

func (s *Server) registerRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.POST("/make-bid", s.MakeBid)
	r.GET("/highest-bid", s.GetHighestBid)

	return r
}
