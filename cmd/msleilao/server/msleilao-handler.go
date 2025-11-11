package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) ConsultAuctions(c *gin.Context) {
	auctions := s.msLeilao.ConsultAuctions()

	c.JSON(http.StatusOK, auctions)
}

func (s *Server) CreateAuction(c *gin.Context) {
	var newAuction struct {
		Descricao string `json:"description"`
		Inicio    string `json:"start"`
		Fim       string `json:"end"`
	}

	if err := c.ShouldBindJSON(&newAuction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad auction format"})
		return
	}

	inicio, err := time.Parse(time.RFC3339, newAuction.Inicio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid start date format",
			"details": "Expected ISO 8601 format (e.g., 2024-01-15T10:00:00Z)",
		})
		return
	}

	fim, err := time.Parse(time.RFC3339, newAuction.Fim)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid end date format",
			"details": "Expected ISO 8601 format (e.g., 2024-01-15T12:00:00Z)",
		})
		return
	}

	if err := s.msLeilao.CreateAuction(newAuction.Descricao, inicio, fim); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("error creating auction: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
