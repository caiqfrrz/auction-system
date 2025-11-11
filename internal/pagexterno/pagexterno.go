package pagexterno

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// Estruturas para request e response
type PaymentRequest struct {
	Amount      float64           `json:"amount"`
	Currency    string            `json:"currency"`
	CallbackURL string            `json:"callback_url"`
	AuctionID   string            `json:"auction_id"`
	WinnerID    string            `json:"winner_id"`
}

type PaymentResponse struct {
	PaymentLink   string `json:"payment_link"`
	TransactionID string `json:"transaction_id"`
}

// Estrutura para notificação de status
type PaymentStatusWebhook struct {
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"` // "approved" | "rejected"
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
	log.Printf("pagexterno received payment request: %+v", req)

	// Simula criação de link e id de transação
	txID := generateTransactionID()
	paymentLink := "https://pay.example.com/" + txID

	resp := PaymentResponse{
		PaymentLink:   paymentLink,
		TransactionID: txID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// processamento assíncrono
	go func() {
		time.Sleep(1 * time.Second) // simula tempo de processamento
		status := "approved"
		if rand.Intn(2) == 0 {
			status = "rejected"
		}
		notify := PaymentStatusWebhook{
			TransactionID: txID,
			Status:        status,
			AuctionID:     req.AuctionID,
			WinnerID:      req.WinnerID,
			Amount:        req.Amount,
		}
		sendPaymentStatusWebhook(req.CallbackURL, notify)
	}()
}

func sendPaymentStatusWebhook(targetURL string, payload PaymentStatusWebhook) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(targetURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status: %s", resp.Status)
	}

	fmt.Println("Webhook sent successfully!")
	return nil
}

func generateTransactionID() string {
	return "tx" + time.Now().Format("20060102150405") + fmt.Sprintf("%04d", rand.Intn(10000))
}

func main() {
    target := "http://localhost:8081/payment-status"

    payload := PaymentStatusWebhook{
        TransactionID: "txn_12345",
        Status:        "approved",
        AuctionID:     "auc_98765",
        WinnerID:      "user_4321",
        Amount:        249.99,
    }

    if err := sendPaymentStatusWebhook(target, payload); err != nil {
        fmt.Println("Error:", err)
    }
}