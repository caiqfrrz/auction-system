package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (s *Server) CancelInterest(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}
