package pagexterno

import (
    "encoding/json"
    "log"
    "math/rand"
    "net/http"
    "time"
)

// Estruturas para request e response
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

// Estrutura para notificação de status
type PaymentStatusWebhook struct {
    TransactionID string            `json:"transaction_id"`
    Status        string            `json:"status"` // "approved" | "rejected"
    AuctionID     string            `json:"auction_id"`
    WinnerID      string            `json:"winner_id"`
    Amount        float64           `json:"amount"`
    Customer      map[string]string `json:"customer"`
}

// Start inicia o servidor HTTP do sistema externo de pagamento
func Start(addr string) {
    http.HandleFunc("/payments", handlePayment)
    log.Printf("pagexterno listening on %s", addr)
    if err := http.ListenAndServe(addr, nil); err != nil {
        log.Fatalf("pagexterno server error: %v", err)
    }
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
        time.Sleep(3 * time.Second) // simula tempo de processamento
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
            Customer:      req.Customer,
        }
        sendWebhook(req.CallbackURL, notify)
    }()
}

func sendWebhook(callbackURL string, payload PaymentStatusWebhook) {
    b, _ := json.Marshal(payload)
    resp, err := http.Post(callbackURL, "application/json", bytes.NewReader(b))
    if err != nil {
        log.Printf("pagexterno webhook error: %v", err)
        return
    }
    defer resp.Body.Close()
    log.Printf("pagexterno sent webhook to %s, status: %s", callbackURL, resp.Status)
}

func generateTransactionID() string {
    return "tx" + time.Now().Format("20060102150405") + fmt.Sprintf("%04d", rand.Intn(10000))
}