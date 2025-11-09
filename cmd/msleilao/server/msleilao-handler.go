package server

import (
	"auction-system/internal/msleilao"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) ConsultAuctions(c *gin.Context) {
	auctions := s.msLeilao.ConsultAuctions()

	c.JSON(http.StatusOK, auctions)
}

func (s *Server) CreateAuction(c *gin.Context) {
	var newAuction msleilao.Auction

	if err := c.ShouldBindJSON(&newAuction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad auction format"})
		return
	}

	if err := s.msLeilao.CreateAuction(newAuction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("error creating auction: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
