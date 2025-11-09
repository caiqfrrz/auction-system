package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateAuction(c *gin.Context) {
	createAuctionReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/create-auction", s.paymentHost), c.Request.Body)

	createAuctionResp, err := http.DefaultClient.Do(createAuctionReq)
	if err != nil || createAuctionResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to create auction: %s", createAuctionResp.Body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}