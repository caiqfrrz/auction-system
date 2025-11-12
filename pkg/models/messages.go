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

type ClienteRegistrado struct {
	UserID    string `json:"user_id"`
	PublicKey string `json:"public_key"`
}
