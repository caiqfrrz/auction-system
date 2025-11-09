package server

import (
	"auction-system/pkg/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) MakeBid(c *gin.Context) {
	var bid models.LanceRealizado

	if err := c.ShouldBindJSON(bid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad bid format"})
		return
	}

	auctions := s.msLance.MakeBid(bid)

	c.JSON(http.StatusOK, auctions)
}
