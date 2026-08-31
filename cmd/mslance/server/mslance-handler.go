package server

import (
	"auction-system/pkg/models"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) MakeBid(c *gin.Context) {
	var bidReq struct {
		UserID   string `json:"user_id"`
		LeilaoID string `json:"leilao_id"`
		Valor    string `json:"valor"`
	}

	if err := c.ShouldBindJSON(&bidReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad bid format"})
		return
	}

	valueNum, err := strconv.ParseFloat(bidReq.Valor, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "value must be a number"})
		return
	}

	bid := models.LanceRealizado{
		UserID:   bidReq.UserID,
		LeilaoID: bidReq.LeilaoID,
		Valor:    valueNum,
	}

	auctions := s.msLance.MakeBid(bid)

	c.JSON(http.StatusOK, auctions)
}

func (s *Server) GetHighestBid(c *gin.Context) {
	auctionID := c.Query("auctionId")
	if auctionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "auctionId query parameter is required",
		})
		return
	}

	highestBid, err := s.msLance.GetHighestBid(auctionID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "failed to retrieve highest bid"

		if err.Error() == fmt.Sprintf("leilão %s não encontrado", auctionID) {
			statusCode = http.StatusNotFound
			errorMsg = "auction not found"
		}

		c.JSON(statusCode, gin.H{
			"error":   errorMsg,
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auction_id":  auctionID,
		"highest_bid": highestBid,
	})
}
