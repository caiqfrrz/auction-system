package server

import (
	"auction-system/internal/gateway/sse"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

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
	createAuctionReq, err := http.NewRequest("POST", fmt.Sprintf("http://%s/create-auction", s.msLeilaoHost), c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create request: %v", err)})
		return
	}

	createAuctionReq.Header.Set("Content-Type", "application/json")

	createAuctionResp, err := http.DefaultClient.Do(createAuctionReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get response: %s", err.Error())})
		return
	}
	defer createAuctionResp.Body.Close()

	body, _ := io.ReadAll(createAuctionResp.Body)

	if createAuctionResp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(body, &errorResponse); err == nil {
			c.JSON(createAuctionResp.StatusCode, errorResponse)
		} else {
			c.JSON(createAuctionResp.StatusCode, gin.H{"error": string(body)})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) ConsultAuctions(c *gin.Context) {
	consultAuctionsReq, err := http.NewRequest("GET", fmt.Sprintf("http://%s/consult-auctions", s.msLeilaoHost), c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error trying to create req: %s", err.Error())})
	}

	consultAuctionsResp, err := http.DefaultClient.Do(consultAuctionsReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get response: %s", err.Error())})
		return
	}
	defer consultAuctionsResp.Body.Close()

	if consultAuctionsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(consultAuctionsResp.Body)
		c.JSON(consultAuctionsResp.StatusCode, gin.H{"error": string(body)})
		return
	}

	var auctions interface{}
	if err := json.NewDecoder(consultAuctionsResp.Body).Decode(&auctions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
		return
	}

	c.JSON(http.StatusOK, auctions)
}

func (s *Server) PlaceBid(c *gin.Context) {
	makeBidReq, err := http.NewRequest("POST", fmt.Sprintf("http://%s/make-bid", s.msLanceHost), c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create request: %v", err)})
		return
	}

	makeBidReq.Header.Set("Content-Type", "application/json")

	makeBidResp, err := http.DefaultClient.Do(makeBidReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get response: %s", err.Error())})
		return
	}
	defer makeBidResp.Body.Close()

	body, _ := io.ReadAll(makeBidResp.Body)

	if makeBidResp.StatusCode != http.StatusOK {
		c.JSON(makeBidResp.StatusCode, gin.H{"error": string(body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) RegisterInterest(c *gin.Context) {
	v, ok := c.Get("client")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "client not found in context"})
		return
	}

	client, ok := v.(sse.Client)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid client type"})
		return
	}

	log.Printf("🔌 Cliente %s iniciou stream no leilão %d", client.ID, client.LeilaoID)

	// Iniciar stream
	c.Stream(func(w io.Writer) bool {
		select {
		case notification, ok := <-client.Channel:
			if !ok {
				log.Printf("❌ Canal fechado para cliente %s", client.ID)
				return false
			}

			log.Printf("📤 Enviando %s para cliente %s", notification.Type, client.ID)

			// Enviar notificação para o cliente
			c.SSEvent(string(notification.Type), notification)

			log.Printf("✅ Evento %s enviado", notification.Type)
			return true

		case <-c.Request.Context().Done():
			log.Printf("⚠️  Contexto cancelado para cliente %s: %v", client.ID, c.Request.Context().Err())
			return false
		}
	})

	log.Printf("🔌 Stream encerrado para cliente %s", client.ID)
}

func (s *Server) CancelInterest(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}
