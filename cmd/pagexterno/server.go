package server

import (
	"auction-system/internal/pagexterno"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Server struct {
	msPagamento *pagexterno.Pagexterno
}

func NewServer(ch *amqp.Channel) *http.Server {
	msPagamento := mspagamento.NewMsPagamento(ch)
	msPagamento.Start()

	NewServer := &Server{
		msPagamento: msPagamento,
	}

	server := &http.Server{
		Addr:         ":8083",
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

	r.POST("/submit-payment-data", s.SubmitPaymentData)

	return r
}
