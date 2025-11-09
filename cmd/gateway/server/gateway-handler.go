package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateAuction(c *gin.Context) {
	createAuctionReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/create-auction", s.msLeilaoHost), c.Request.Body)

	createAuctionResp, err := http.DefaultClient.Do(createAuctionReq)
	if err != nil || createAuctionResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to create auction: %s", createAuctionResp.Body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) ConsultAuctions(c *gin.Context) {
	consultAuctionsReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/consult-auctions", s.msLeilaoHost), c.Request.Body)

	consultAuctionsResp, err := http.DefaultClient.Do(consultAuctionsReq)
	if err != nil || consultAuctionsResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to consult auctions: %s", consultAuctionsResp.Body)})
		return
	}

	c.JSON(http.StatusOK, consultAuctionsResp.Body)
}

func (s *Server) PlaceBid(c *gin.Context) {
	makeBidReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/make-bid", s.msLanceHost), c.Request.Body)

	makeBidResp, err := http.DefaultClient.Do(makeBidReq)
	if err != nil || makeBidResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to make bid: %s", makeBidResp.Body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (s *Server) RegisterInterest(c *gin.Context) {

}

func (s *Server) CancelInterest(c *gin.Context) {

}
