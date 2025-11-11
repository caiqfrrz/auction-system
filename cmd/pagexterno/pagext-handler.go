package server

import (
	"auction-system/internal/pagexterno"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) SubmitPaymentData(c *gin.Context) {
	var newPayData pagexterno.PaymentRequest

	if err := c.ShouldBindJSON(&newPayData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad auction format"})
		return
	}

	if err := s.pagexterno.CreateAuction(newPayData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("error creating auction: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})

	SubmitPaymentDataReq, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/create-auction", s.paymentHost), c.Request.Body)

	SubmitPaymentDataResp, err := http.DefaultClient.Do(SubmitPaymentDataReq)
	if err != nil || SubmitPaymentDataResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to create auction: %s", SubmitPaymentDataResp.Body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}