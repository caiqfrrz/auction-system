package mspagamento

import (
	"auction-system/pkg/models"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type MsPagamento struct {
	externalPayURL  string
	publicURL       string
	onPaymentLink   func(models.LinkPagamento)
	onPaymentStatus func(models.StatusPagamento)
}

type PaymentRequest struct {
	Amount      float64           `json:"amount"`
	Currency    string            `json:"currency"`
	Customer    map[string]string `json:"customer"`
	CallbackURL string            `json:"callback_url"`
	AuctionID   string            `json:"auction_id"`
	WinnerID    string            `json:"winner_id"`
}

type PaymentResponse struct {
	PaymentLink   string `json:"payment_link"`
	TransactionID string `json:"transaction_id"`
}

type PaymentStatusWebhook struct {
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"`
	AuctionID     string  `json:"auction_id"`
	WinnerID      string  `json:"winner_id"`
	Amount        float64 `json:"amount"`
}

func NewMsPagamento(publicURL, externalPayURL string) *MsPagamento {
	return &MsPagamento{
		externalPayURL: externalPayURL,
		publicURL:      publicURL,
	}
}

// Registrar callbacks
func (m *MsPagamento) SetPaymentCallbacks(
	onLink func(models.LinkPagamento),
	onStatus func(models.StatusPagamento),
) {
	m.onPaymentLink = onLink
	m.onPaymentStatus = onStatus
}

func (m *MsPagamento) SubmitPaymentData(leilao models.LeilaoVencedor) error {
	log.Printf("[MS PAGAMENTO] Processando vencedor: %+v", leilao)

	req := PaymentRequest{
		Amount:      leilao.Valor,
		Currency:    "BRL",
		Customer:    map[string]string{"id": leilao.UserID},
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

	linkPagamento := models.LinkPagamento{
		UserID:        leilao.UserID,
		PaymentLink:   payResp.PaymentLink,
		TransactionID: payResp.TransactionID,
		AuctionID:     leilao.LeilaoID,
	}

	log.Printf("Link de pagamento recebido: %s", payResp.PaymentLink)

	// Notificar via callback
	if m.onPaymentLink != nil {
		m.onPaymentLink(linkPagamento)
	}

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

	statusPagamento := models.StatusPagamento{
		TransactionID: payload.TransactionID,
		Status:        payload.Status,
		AuctionID:     payload.AuctionID,
		WinnerID:      payload.WinnerID,
		Amount:        payload.Amount,
	}

	// Notificar via callback
	if m.onPaymentStatus != nil {
		m.onPaymentStatus(statusPagamento)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received"))
}

func (m *MsPagamento) StartWebhookServer(addr string) error {
	http.HandleFunc("/payment-status", m.webhookHandler)
	log.Printf("[MS PAGAMENTO] Servidor webhook ouvindo em %s", addr)
	return http.ListenAndServe(addr, nil)
}
