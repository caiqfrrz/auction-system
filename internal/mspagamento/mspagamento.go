package mspagamento

import (
	"auction-system/pkg/models"
	amqp "github.com/rabbitmq/amqp091-go"
	"auction-system/internal/rabbitmq"
	"encoding/json"
	"log"
	"net/http"
	"bytes"
	"fmt"	
)

type MsPagamento struct {
    ch             *amqp.Channel
    externalPayURL string
    publicURL      string // usado para montar callback (ex: http://host:port)
    queueName      string
    httpAddr       string // endereco para expor webhook (ex ":8081")
}

func NewMsPagamento(ch *amqp.Channel, externalPayURL, publicURL, queueName, httpAddr string) *MsPagamento {
	return &MsPagamento{
		ch:             ch,
		externalPayURL: externalPayURL,
		publicURL:      publicURL,
		queueName:      queueName,
		httpAddr:       ":8081",
	}
}

// Inicializa a exchange e faz o binding das filas
func (m *MsPagamento) DeclareExchangeAndQueues() {
	rabbitmq.DeclareExchange(m.ch, "ms_pagamentos", "topic")

	rabbitmq.DeclareQueue(m.ch, "leilao_vencedor")
	rabbitmq.BindQueueToExchange(m.ch, "leilao_vencedor", "leilao_vencedor", "ms_pagamentos")

	rabbitmq.DeclareQueue(m.ch, "link_pagamento")
	rabbitmq.BindQueueToExchange(m.ch, "link_pagamento", "link_pagamento", "ms_pagamentos")
}

func(m* MsPagamento) Listenleilao_vencedor(){
	msgs, _ := m.ch.Consume("leilao_vencedor", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var leilao models.LeilaoVencedor
			if err := json.Unmarshal(d.Body, &leilao); err == nil {
				// uma requisição REST ao sistema externo de pagamentos enviando os dados do pagamento 
				// (valor, moeda, informações do cliente) e, então, receberá um link de
				//pagamento que será publicado em link_pagamento.
				}
				log.Printf("requisição REST payment ao sistema de pagamento enviada: usuário %s (valor %s)", leilao.UserID, leilao.Valor)
			}
		}()
}
		