package server

import (
	"auction-system/internal/gateway/sse"
	lancePb "auction-system/proto/lance"
	leilaoPb "auction-system/proto/leilao"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// mandatory headers for sse
func HeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Next()
	}
}

func (s *Server) CreateAuction(c *gin.Context) {
	var req struct {
		Titulo          string  `json:"titulo" binding:"required"`
		Descricao       string  `json:"descricao" binding:"required"`
		ValorInicial    float64 `json:"valor_inicial" binding:"required,gt=0"`
		DuracaoSegundos int64   `json:"duracao_segundos" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// gRPC call
	resp, err := s.grpcClients.LeilaoClient.CreateAuction(ctx, &leilaoPb.CreateAuctionRequest{
		Titulo:          req.Titulo,
		Descricao:       req.Descricao,
		ValorInicial:    req.ValorInicial,
		DuracaoSegundos: req.DuracaoSegundos,
	})

	if err != nil {
		log.Printf("Error calling CreateAuction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create auction"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.Error})
		return
	}

	// Notificar MSLance via gRPC
	go func() {
		ctx := context.Background()
		_, err := s.grpcClients.LanceClient.NotifyAuctionStarted(ctx, &lancePb.AuctionStartedNotification{
			LeilaoId: resp.LeilaoId,
			Duracao:  req.DuracaoSegundos,
		})
		if err != nil {
			log.Printf("Error notifying MSLance: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"leilao_id": resp.LeilaoId})
}

func (s *Server) ConsultAuctions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.grpcClients.LeilaoClient.ConsultAuctions(ctx, &leilaoPb.ConsultAuctionsRequest{})
	if err != nil {
		log.Printf("Error calling ConsultAuctions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch auctions"})
		return
	}

	// Converter de protobuf para JSON
	var auctions []map[string]interface{}
	for _, a := range resp.Auctions {
		auctions = append(auctions, map[string]interface{}{
			"id":             a.Id,
			"titulo":         a.Titulo,
			"descricao":      a.Descricao,
			"valor_inicial":  a.ValorInicial,
			"tempo_restante": a.TempoRestante,
			"active":         a.Active,
		})
	}

	c.JSON(http.StatusOK, gin.H{"auctions": auctions})
}

func (s *Server) PlaceBid(c *gin.Context) {
	var req struct {
		UserID   string  `json:"user_id" binding:"required"`
		LeilaoID string  `json:"leilao_id" binding:"required"`
		Valor    float64 `json:"valor" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.grpcClients.LanceClient.MakeBid(ctx, &lancePb.MakeBidRequest{
		UserId:   req.UserID,
		LeilaoId: req.LeilaoID,
		Valor:    req.Valor,
	})

	if err != nil {
		log.Printf("Error calling MakeBid: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to place bid"})
		return
	}

	if !resp.Success {
		// Lance inválido - enviar via SSE
		notification := sse.Notification{
			Type:      sse.LanceInvalidado,
			ClienteID: req.UserID,
			LeilaoID:  mustAtoi(req.LeilaoID),
			Data: map[string]interface{}{
				"motivo": resp.Error,
			},
		}
		s.eventStream.Message <- notification

		c.JSON(http.StatusBadRequest, gin.H{"error": resp.Error})
		return
	}

	// Lance válido - broadcast via SSE
	notification := sse.Notification{
		Type:     sse.LanceValidado,
		LeilaoID: mustAtoi(req.LeilaoID),
		Data: map[string]interface{}{
			"user_id": req.UserID,
			"valor":   req.Valor,
		},
	}
	s.eventStream.Message <- notification

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) GetHighestBid(c *gin.Context) {
	auctionID := c.Query("auctionId")
	if auctionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auctionId required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.grpcClients.LanceClient.GetHighestBid(ctx, &lancePb.GetHighestBidRequest{
		LeilaoId: auctionID,
	})

	if err != nil {
		log.Printf("Error calling GetHighestBid: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get highest bid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auction_id":  resp.LeilaoId,
		"highest_bid": resp.HighestBid,
	})
}

func (s *Server) RegisterInterest(c *gin.Context) {
	v, ok := c.Get("client")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "client not found"})
		return
	}

	client, ok := v.(sse.Client)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid client type"})
		return
	}

	log.Printf("Cliente %s iniciou stream no leilão %d", client.ID, client.LeilaoID)

	c.Stream(func(w io.Writer) bool {
		select {
		case notification := <-client.Channel:
			data, _ := json.Marshal(notification.Data)
			fmt.Fprintf(w, "event: %s\n", notification.Type)
			fmt.Fprintf(w, "data: %s\n\n", data)
			return true

		case <-c.Request.Context().Done():
			log.Printf("Cliente %s desconectou do leilão %d", client.ID, client.LeilaoID)
			return false
		}
	})
}

func (s *Server) CancelInterest(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func mustAtoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
