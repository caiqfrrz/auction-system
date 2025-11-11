package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

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

func handlePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	log.Printf("[PAGEXTERNO] Nova requisição de pagamento: %+v", req)

	txID := generateTransactionID()
	paymentLink := fmt.Sprintf("https://pay.example.com/%s", txID)

	resp := PaymentResponse{
		PaymentLink:   paymentLink,
		TransactionID: txID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// processamento assíncrono (simulado)
	go func() {
		time.Sleep(3 * time.Second)
		status := []string{"approved", "rejected"}[rand.Intn(2)]
		notify := PaymentStatusWebhook{
			TransactionID: txID,
			Status:        status,
			AuctionID:     req.AuctionID,
			WinnerID:      req.WinnerID,
			Amount:        req.Amount,
		}
		if err := sendPaymentStatusWebhook(req.CallbackURL, notify); err != nil {
			log.Println("Erro ao enviar webhook:", err)
		}
	}()
}

func sendPaymentStatusWebhook(targetURL string, payload PaymentStatusWebhook) error {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(targetURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}

	log.Printf("[PAGEXTERNO] Webhook enviado com sucesso para %s", targetURL)
	return nil
}

func generateTransactionID() string {
	return fmt.Sprintf("tx-%d", time.Now().UnixNano())
}

func main() {
	rand.Seed(time.Now().UnixNano())
	http.HandleFunc("/payment", handlePayment)
	log.Println("[PAGEXTERNO] Servidor ouvindo em :8085")
	http.ListenAndServe(":8085", nil)
}
