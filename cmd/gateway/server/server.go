package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

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
	r.POST("/create-auction", s.CreateAuction)
	r.POST("/make-bid", s.PlaceBid)
	r.POST("/register-interest", s.RegisterInterest)
	r.POST("/cancel-interest", s.CancelInterest)

	return r
}
