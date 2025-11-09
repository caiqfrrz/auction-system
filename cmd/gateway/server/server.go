package server

import (
	"auction-system/internal/gateway"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	port         int
	msLanceHost  string
	msLeilaoHost string
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	msLeilao := os.Getenv("MSLEILAO_HOST")
	msLance := os.Getenv("MSLANCE_HOST")

	NewServer := &Server{
		port:         port,
		msLanceHost:  msLance,
		msLeilaoHost: msLeilao,
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
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

	r.GET("/consult-auctions", gateway.ConsultAuctions)
	r.POST("/create-auction", gateway.CreateAuction)
	r.POST("/make-bid", gateway.PlaceBid)
	r.POST("/register-interest", gateway.RegisterInterest)
	r.POST("/cancel-interest", gateway.CancelInterest)

	return r
}
