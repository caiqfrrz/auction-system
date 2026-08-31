package server

import (
	"auction-system/internal/msleilao"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Server struct {
	msLeilao *msleilao.MsLeilao
}

func NewServer(ch *amqp.Channel) *http.Server {
	msLeilao := msleilao.NewMsLeilao(ch)
	msLeilao.Start()

	NewServer := &Server{
		msLeilao: msLeilao,
	}

	server := &http.Server{
		Addr:         ":8081",
		Handler:      NewServer.registerRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

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

	r.GET("/consult-auctions", s.ConsultAuctions)
	r.POST("/create-auction", s.CreateAuction)

	return r
}
