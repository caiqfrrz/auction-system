package server

import (
	"auction-system/pkg/models"
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
