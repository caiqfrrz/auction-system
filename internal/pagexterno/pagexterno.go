package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

// Structs

type PaymentRequest struct {
	Amount      float64           `json:"amount"`
    Currency    string            `json:"currency"`
    Customer    map[string]string `json:"customer"`
    CallbackURL string            `json:"callback_url"`
    AuctionID   string            `json:"auction_id"`
    WinnerID    string            `json:"winner_id"`
	//LinkCB      string            `json:"link_callback"`
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

type PaymentStore struct {
	sync.RWMutex
	data map[string]PaymentRequest
}

func NewPaymentStore() *PaymentStore {
	return &PaymentStore{data: make(map[string]PaymentRequest)}
}

func (ps *PaymentStore) Set(id string, req PaymentRequest) {
	ps.Lock()
	defer ps.Unlock()
	ps.data[id] = req
}

func (ps *PaymentStore) Get(id string) (PaymentRequest, bool) {
	ps.RLock()
	defer ps.RUnlock()
	req, ok := ps.data[id]
	return req, ok
}

func (ps *PaymentStore) Delete(id string) {
	ps.Lock()
	defer ps.Unlock()
	delete(ps.data, id)
}

func main() {
	ps := NewPaymentStore() // single shared instance

	mux := http.NewServeMux()

	mux.HandleFunc("/payment", handlePayment(ps))
	mux.HandleFunc("/pay/", handlePaymentPage(ps))
	mux.HandleFunc("/complete/", handlePaymentAction(ps))

	log.Println("[PAGEXTERNO] Listening on :8085")
	http.ListenAndServe(":8085", mux)
}

func handlePayment(ps *PaymentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		ps.Set(txID, req)

		paymentLink := fmt.Sprintf("http://localhost:8085/pay/%s", txID)
		resp := PaymentResponse{
			PaymentLink:   paymentLink,
			TransactionID: txID,
		}

		// send response instantly to MS Pagamento
		// if req.LinkCB != "" {
		// 	go func() {
		// 		sendPaymentLinkCallback(req.LinkCB, resp)
		// 	}()
		// }

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		log.Printf("[PAGEXTERNO] Nova transação %s criada (%.2f %s) \n%s", txID, req.Amount, req.Currency, paymentLink)
	}
}

func handlePaymentPage(ps *PaymentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txID := r.URL.Path[len("/pay/"):]
		req, ok := ps.Get(txID)
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
}

func handlePaymentAction(ps *PaymentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txID := r.URL.Path[len("/complete/"):]
		status := r.FormValue("status")

		req, ok := ps.Get(txID)
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

		// Asynchronous webhook send
		go func() {
			if err := sendPaymentStatusWebhook(req.CallbackURL, notify); err != nil {
				log.Println("Erro ao enviar webhook:", err)
			}
		}()

		fmt.Fprintf(w, "<h3>Pagamento %s com sucesso!</h3>", status)
		ps.Delete(txID)
	}
}

// Helpers

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

func sendPaymentLinkCallback(targetURL string, payload PaymentResponse) {
	body, _ := json.Marshal(payload)
	_, err := http.Post(targetURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Erro ao enviar PaymentResponse:", err)
		return
	}
	log.Printf("[PAGEXTERNO] Payment link enviado → %s", targetURL)
}

func generateTransactionID() string {
	return fmt.Sprintf("tx-%d", time.Now().UnixNano())
}
