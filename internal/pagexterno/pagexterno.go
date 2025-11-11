package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// In-memory store of pending payments
var payments = struct {
	sync.RWMutex
	data map[string]PaymentRequest
}{data: make(map[string]PaymentRequest)}

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

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/payment", handlePayment)         // POST from MS Pagamento
	http.HandleFunc("/pay/", handlePaymentPage)        // GET page for user to click Pay/Cancel
	http.HandleFunc("/complete/", handlePaymentAction) // POST from HTML form

	log.Println("[PAGEXTERNO] Servidor ouvindo em :8085")
	http.ListenAndServe(":8085", nil)
}

// Called by MS Pagamento
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

	txID := generateTransactionID()
	payments.Lock()
	payments.data[txID] = req
	payments.Unlock()

	paymentLink := fmt.Sprintf("http://localhost:8085/pay/%s", txID)

	resp := PaymentResponse{
		PaymentLink:   paymentLink,
		TransactionID: txID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	log.Printf("[PAGEXTERNO] Nova transação criada: %s (%.2f %s) \n%s", txID, req.Amount, req.Currency, paymentLink)
}

// Renders the payment page (HTML)
func handlePaymentPage(w http.ResponseWriter, r *http.Request) {
	txID := r.URL.Path[len("/pay/"):]
	payments.RLock()
	req, ok := payments.data[txID]
	payments.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	tmpl := `
	<html>
		<head><title>Pagamento {{.TxID}}</title></head>
		<body style="font-family:sans-serif; text-align:center; margin-top:40px;">
			<h2>Pagamento do Leilão {{.AuctionID}}</h2>
			<p><b>Valor:</b> R$ {{printf "%.2f" .Amount}}</p>
			<p><b>Cliente:</b> {{.WinnerID}}</p>
			<form action="/complete/{{.TxID}}" method="POST" style="margin-top:20px;">
				<button name="status" value="approved" style="padding:10px 20px; background:green; color:white; border:none;">Pagar</button>
				<button name="status" value="rejected" style="padding:10px 20px; background:red; color:white; border:none;">Cancelar</button>
			</form>
		</body>
	</html>`

	t := template.Must(template.New("payment").Parse(tmpl))
	t.Execute(w, map[string]interface{}{
		"TxID":      txID,
		"AuctionID": req.AuctionID,
		"Amount":    req.Amount,
		"WinnerID":  req.WinnerID,
	})
}

// Called when user clicks Pay/Cancel
func handlePaymentAction(w http.ResponseWriter, r *http.Request) {
	txID := r.URL.Path[len("/complete/"):]
	status := r.FormValue("status")

	payments.RLock()
	req, ok := payments.data[txID]
	payments.RUnlock()

	if !ok {
		http.Error(w, "transaction not found", http.StatusNotFound)
		return
	}

	notify := PaymentStatusWebhook{
		TransactionID: txID,
		Status:        status,
		AuctionID:     req.AuctionID,
		WinnerID:      req.WinnerID,
		Amount:        req.Amount,
	}

	// Send webhook back to MS Pagamento
	if err := sendPaymentStatusWebhook(req.CallbackURL, notify); err != nil {
		log.Println("Erro ao enviar webhook:", err)
		http.Error(w, "Erro ao enviar webhook", 500)
		return
	}

	fmt.Fprintf(w, "<h3>Pagamento %s com sucesso!</h3>", status)
	delete(payments.data, txID)
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

	log.Printf("[PAGEXTERNO] Webhook enviado (%s) → %s", payload.Status, targetURL)
	return nil
}

func generateTransactionID() string {
	return fmt.Sprintf("tx-%d", time.Now().UnixNano())
}
