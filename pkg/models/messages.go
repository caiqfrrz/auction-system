package models

import "time"

type LeilaoIniciado struct {
	ID         string    `json:"id"`
	Descricao  string    `json:"descricao"`
	DataInicio time.Time `json:"data_inicio"`
	DataFim    time.Time `json:"data_fim"`
}

type LanceRealizado struct {
	LeilaoID   string  `json:"leilao_id"`
	UserID     string  `json:"user_id"`
	Valor      float64 `json:"valor"`
	Assinatura string  `json:"assinatura"`
}

type LanceValidado struct {
	LeilaoID string  `json:"leilao_id"`
	UserID   string  `json:"user_id"`
	Valor    float64 `json:"valor"`
}

type LeilaoVencedor struct {
	LeilaoID string  `json:"leilao_id"`
	UserID   string  `json:"user_id"`
	Valor    float64 `json:"valor"`
}
