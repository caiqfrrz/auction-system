package mspagamento

import (
	"auction-system/pkg/models"
	"auction-system/pkg/rabbitmq"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
	//LinkCB      string            `json:"link_callback"`
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
	rabbitmq.DeclareExchange(m.ch, "leilao_events", "topic")

	rabbitmq.DeclareQueue(m.ch, "mspag_leilao_vencedor")
	rabbitmq.BindQueueToExchange(m.ch, "mspag_leilao_vencedor", "leilao.vencedor", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "link_pagamento")
	rabbitmq.BindQueueToExchange(m.ch, "link_pagamento", "link.pagamento", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "status_pagamento")
	rabbitmq.BindQueueToExchange(m.ch, "status_pagamento", "status.pagamento", "leilao_events")
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
		Amount:      leilao.Valor,
		Currency:    "BRL",
		Customer:    map[string]string{"id": leilao.UserID},
		CallbackURL: fmt.Sprintf("%s/payment-status", m.publicURL),
		AuctionID:   leilao.LeilaoID,
		WinnerID:    leilao.UserID,
		//LinkCB:      fmt.Sprintf("%s/payment-link", m.publicURL),
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

	var linkPagamento = models.LinkPagamento{
		UserID:        leilao.UserID,
		PaymentLink:   payResp.PaymentLink,
		TransactionID: payResp.TransactionID,
		AuctionID:     leilao.LeilaoID,
	}

	// Publica o link no RabbitMQ
	msgBody, _ := json.Marshal(linkPagamento)
	if err := rabbitmq.PublishToExchange(m.ch, "leilao_events", "link.pagamento", msgBody); err != nil {
		return fmt.Errorf("erro ao publicar link_pagamento: %w", err)
	}

	log.Printf("Link de pagamento publicado: %s", payResp.PaymentLink)
	return nil
}

func (m *MsPagamento) webhookHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Entrou no webhookHandler")
	var payload PaymentStatusWebhook
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("[WEBHOOK] Status recebido: %+v", payload)

	var statusPagamento = models.StatusPagamento{
		TransactionID: payload.TransactionID,
		Status:        payload.Status,
		AuctionID:     payload.AuctionID,
		WinnerID:      payload.WinnerID,
		Amount:        payload.Amount,
	}

	// Publica o evento status_pagamento
	msgBody, _ := json.Marshal(statusPagamento)
	if err := m.ch.Publish("leilao_events", "status.pagamento", false, false,
		amqp.Publishing{ContentType: "application/json", Body: msgBody}); err != nil {
		log.Println("Erro ao publicar status_pagamento:", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received"))
}

// func (m *MsPagamento) paymentLinkHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Println("Entrou no paymentLinkHandler")
// 	var resp PaymentResponse
// 	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
// 		http.Error(w, "invalid JSON", http.StatusBadRequest)
// 		return
// 	}
// 	defer r.Body.Close()

// 	log.Printf("[MS PAGAMENTO] Link recebido: %s (tx=%s)", resp.PaymentLink, resp.TransactionID)

// 	// Publish to queue "link_pagamento"
// 	body, _ := json.Marshal(resp)
// 	if err := m.ch.Publish("leilao_events", "link.pagamento", false, false,
// 		amqp.Publishing{ContentType: "application/json", Body: body}); err != nil {
// 		log.Println("Erro ao publicar link_pagamento:", err)
// 	}

// 	w.WriteHeader(http.StatusOK)
// }

func (m *MsPagamento) sendTestLeilaoVencedor() {
	lv := models.LeilaoVencedor{
		LeilaoID: "test-auction-1",
		UserID:   "test-winner-1",
		Valor:    123.45,
	}
	b, err := json.Marshal(lv)
	if err != nil {
		log.Printf("failed to marshal test leilao_vencedor: %v", err)
		return
	}
	if err := rabbitmq.PublishToExchange(m.ch, "leilao_events", "leilao.vencedor", b); err != nil {
		log.Printf("failed to publish test leilao_vencedor: %v", err)
		return
	}
	log.Printf("[MS PAGAMENTO] Test leilao_vencedor published: %+v", lv)
}

func (m *MsPagamento) Start() {
	m.DeclareExchangeAndQueues()
	m.ListenLeilaoVencedor()

	go func() {
		// pequeno delay para garantir que o consumer esteja registrado
		time.Sleep(100 * time.Millisecond)
		m.sendTestLeilaoVencedor()
	}()

	http.HandleFunc("/payment-status", m.webhookHandler)
	//http.HandleFunc("/payment-link", m.paymentLinkHandler)
	log.Printf("[MS PAGAMENTO] Servidor ouvindo webhook em %s", m.httpAddr)
	http.ListenAndServe(m.httpAddr, nil)
}
