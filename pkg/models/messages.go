package models

import "time"

type LeilaoIniciado struct {
	ID         string    `json:"id"`
	Descricao  string    `json:"descricao"`
	DataInicio time.Time `json:"data_inicio"`
	DataFim    time.Time `json:"data_fim"`
}

type LeilaoFinalizado struct {
	ID        string `json:"id"`
	Descricao string `json:"descricao"`
}

type LanceRealizado struct {
	LeilaoID string  `json:"leilao_id"`
	UserID   string  `json:"user_id"`
	Valor    float64 `json:"valor"`
}

type LanceValidado struct {
	LeilaoID string  `json:"leilao_id"`
	UserID   string  `json:"user_id"`
	Valor    float64 `json:"valor"`
}

type LanceInvalidado struct {
	LeilaoID string  `json:"leilao_id"`
	UserID   string  `json:"user_id"`
	Valor    float64 `json:"valor"`
	Motivo   string  `json:"motivo"`
}

type LeilaoVencedor struct {
	LeilaoID string  `json:"leilao_id"`
	UserID   string  `json:"user_id"`
	Valor    float64 `json:"valor"`
}

type StatusPagamento struct {
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"` // "approved" | "rejected"
	AuctionID     string  `json:"auction_id"`
	WinnerID      string  `json:"winner_id"`
	Amount        float64 `json:"amount"`
}

type LinkPagamento struct {
	UserID        string `json:"user_id"`
	PaymentLink   string `json:"payment_link"`
	TransactionID string `json:"transaction_id"`
	AuctionID     string `json:"auction_id"`
}
