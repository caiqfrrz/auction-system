package mspagamento

import (
	"auction-system/internal/rabbitmq"
	"auction-system/pkg/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MsPagamento struct {
	ch             *amqp.Channel
	externalPayURL string
	publicURL      string // usado para montar callback (ex: http://host:port)
	queueName      string
	httpAddr       string // endereco para expor webhook (ex ":8081")
}

type PaymentRequest struct {
	Amount      float64           `json:"amount"`
    Currency    string            `json:"currency"`
    Customer    map[string]string `json:"customer"`
    CallbackURL string            `json:"callback_url"`
    AuctionID   string            `json:"auction_id"`
    WinnerID    string            `json:"winner_id"`
}

func NewMsPagamento(ch *amqp.Channel, externalPayURL, publicURL, queueName, httpAddr string) *MsPagamento {
	return &MsPagamento{
		ch:             ch,
		externalPayURL: externalPayURL,
		publicURL:      publicURL,
		queueName:      queueName,
		httpAddr:       ":8084",
	}
}

// Inicializa a exchange e faz o binding das filas
func (m *MsPagamento) DeclareExchangeAndQueues() {
	rabbitmq.DeclareExchange(m.ch, "ms_pagamentos", "topic")

	rabbitmq.DeclareQueue(m.ch, "leilao_vencedor")
	rabbitmq.BindQueueToExchange(m.ch, "leilao_vencedor", "leilao_vencedor", "ms_pagamentos")

	rabbitmq.DeclareQueue(m.ch, "link_pagamento")
	rabbitmq.BindQueueToExchange(m.ch, "link_pagamento", "link_pagamento", "ms_pagamentos")

	rabbitmq.DeclareQueue(m.ch, "status_pagamento,")
	rabbitmq.BindQueueToExchange(m.ch, "status_pagamento", "status_pagamento", "ms_pagamentos")
}

func (m *MsPagamento) Listenleilao_vencedor() {
	msgs, _ := m.ch.Consume("leilao_vencedor", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var leilao models.LeilaoVencedor
			if err := json.Unmarshal(d.Body, &leilao); err == nil {
				m.SubmitPaymentData()
			}
			log.Printf("requisição REST payment ao sistema de pagamento enviada: usuário %s (valor %s)", leilao.UserID, leilao.Valor)
		}
	}()
}
func (m *MsPagamento) Start() {
	m.DeclareExchangeAndQueues()
	m.Listenleilao_vencedor()
}

func (m *MsPagamento) SubmitPaymentData(c *gin.Context) {
	SubmitPaymentDataReq, _ := http.NewRequest("POST", fmt.Sprintf("http://%s/create-auction", m.externalPayURL), c.Request.Body)

	SubmitPaymentDataResp, err := http.DefaultClient.Do(SubmitPaymentDataReq)
	if err != nil || SubmitPaymentDataResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to create auction: %s", SubmitPaymentDataResp.Body)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
