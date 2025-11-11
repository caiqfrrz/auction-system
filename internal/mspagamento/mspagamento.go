package mspagamento

import (
	"auction-system/internal/rabbitmq"
	"auction-system/pkg/models"
	"encoding/json"
	"fmt"
	"bytes"
	"log"
	"net/http"

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

type PaymentStatusWebhook struct {
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"` // "approved" | "rejected"
	AuctionID     string  `json:"auction_id"`
	WinnerID      string  `json:"winner_id"`
	Amount        float64 `json:"amount"`
}

type PaymentResponse struct {
	PaymentLink   string `json:"payment_link"`
	TransactionID string `json:"transaction_id"`
}

func NewMsPagamento(ch *amqp.Channel, externalPayURL, publicURL, queueName, httpAddr string) *MsPagamento {
	return &MsPagamento{
		ch:             ch,
		externalPayURL: externalPayURL,
		publicURL:      publicURL,
		queueName:      queueName,
		httpAddr:       httpAddr,
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

func (m *MsPagamento) ListenLeilaoVencedor() {
	msgs, _ := m.ch.Consume("leilao_vencedor", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var leilao models.LeilaoVencedor
			if err := json.Unmarshal(d.Body, &leilao); err != nil {
				log.Println("Error decoding leilao_vencedor:", err)
				continue
			}

			log.Printf("[MS PAGAMENTO] Recebido vencedor: %+v", leilao)
			if err := m.SubmitPaymentData(leilao); err != nil {
				log.Println("Erro ao enviar pagamento:", err)
			}
		}
	}()
}

func (m *MsPagamento) SubmitPaymentData(leilao models.LeilaoVencedor) error {
	req := PaymentRequest{
		Amount:   leilao.Valor,
		Currency: "BRL",
		Customer: map[string]string{"id": leilao.UserID},
		CallbackURL: fmt.Sprintf("%s/payment-status", m.publicURL),
		AuctionID:   leilao.LeilaoID,
		WinnerID:    leilao.UserID,
	}

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/payment", m.externalPayURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("erro ao chamar sistema de pagamento: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erro no retorno do sistema externo: %s", resp.Status)
	}

	var payResp PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&payResp); err != nil {
		return fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	// Publica o link no RabbitMQ
	msgBody, _ := json.Marshal(payResp)
	if err := m.ch.Publish("ms_pagamentos", "link_pagamento", false, false,
		amqp.Publishing{ContentType: "application/json", Body: msgBody}); err != nil {
		return fmt.Errorf("erro ao publicar link_pagamento: %w", err)
	}

	log.Printf("Link de pagamento publicado: %s", payResp.PaymentLink)
	return nil
}

func (m *MsPagamento) webhookHandler(w http.ResponseWriter, r *http.Request) {
	var payload PaymentStatusWebhook
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("[WEBHOOK] Status recebido: %+v", payload)

	// Publica o evento status_pagamento
	msgBody, _ := json.Marshal(payload)
	if err := m.ch.Publish("ms_pagamentos", "status_pagamento", false, false,
		amqp.Publishing{ContentType: "application/json", Body: msgBody}); err != nil {
		log.Println("Erro ao publicar status_pagamento:", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received"))
}

func (m *MsPagamento) Start() {
	m.DeclareExchangeAndQueues()
	m.ListenLeilaoVencedor()

	http.HandleFunc("/payment-status", m.webhookHandler)
	log.Printf("[MS PAGAMENTO] Servidor ouvindo webhook em %s", m.httpAddr)
	http.ListenAndServe(m.httpAddr, nil)
}