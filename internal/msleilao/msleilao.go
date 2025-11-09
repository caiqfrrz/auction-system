package msleilao

import (
	"auction-system/pkg/models"
	"auction-system/pkg/rabbitmq"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Auction struct {
	ID        string    `json:"id"`
	Descricao string    `json:"descricao"`
	Inicio    time.Time `json:"inicio"`
	Fim       time.Time `json:"fim"`
	Ativo     bool      `json:"ativo"`
}

type MsLeilao struct {
	ch       *amqp.Channel
	auctions []Auction
}

func NewMsLeilao(ch *amqp.Channel) *MsLeilao {
	// now := time.Now()
	auctions := []Auction{
		// {ID: "1", Descricao: "Almoço no RU", Inicio: now.Add(2 * time.Second), Fim: now.Add(50 * time.Second), Ativo: false},
		// {ID: "2", Descricao: "Monalisa", Inicio: now.Add(20 * time.Second), Fim: now.Add(40 * time.Second), Ativo: false},
	}

	return &MsLeilao{ch: ch, auctions: auctions}
}

func (l *MsLeilao) CreateAuction(auction Auction) error {
	now := time.Now()

	if strings.TrimSpace(auction.Descricao) == "" {
		return fmt.Errorf("description cannot be empty")
	}

	if auction.Inicio.Before(now) {
		return fmt.Errorf("start time cannot be in the past")
	}

	if auction.Fim.Before(now) {
		return fmt.Errorf("end time cannot be in the past")
	}

	l.auctions = append(l.auctions, auction)

	return nil
}

func (l *MsLeilao) ConsultAuctions() []Auction {
	return l.auctions
}

func (l *MsLeilao) Start() {
	rabbitmq.DeclareExchange(l.ch, "leilao_events", "topic")
	for i := range l.auctions {
		auction := &l.auctions[i]

		go func(a *Auction) {
			time.Sleep(time.Until(a.Inicio))
			a.Ativo = true
			event := models.LeilaoIniciado{
				ID:         a.ID,
				Descricao:  a.Descricao,
				DataInicio: a.Inicio,
				DataFim:    a.Fim,
			}
			body, _ := json.Marshal(event)
			// Publica na exchange com routing key
			rabbitmq.PublishToExchange(l.ch, "leilao_events", "leilao.iniciado", body)
			log.Printf("Leilão %s iniciado!", a.ID)
		}(auction)

		go func(a *Auction) {
			// aguarda até 1s antes do fim → mensagem 1
			queueName := fmt.Sprintf("leilao_%s", a.ID)
			time.Sleep(time.Until(a.Fim.Add(-3 * time.Second)))
			msg1 := "DOLE UMA!"
			body1, _ := json.Marshal(msg1)
			rabbitmq.PublishToExchange(l.ch, "leilao_events", queueName, body1, "text/plain")

			// aguarda até 1s antes do fim → mensagem 2
			time.Sleep(1 * time.Second)
			msg2 := "DOLE DUAS!"
			body2, _ := json.Marshal(msg2)
			rabbitmq.PublishToExchange(l.ch, "leilao_events", queueName, body2, "text/plain")

			// chegou o fim → mensagem VENDIDO!!
			time.Sleep(1 * time.Second)
			a.Ativo = false
			msg3 := "VENDIDO!!!!"
			body3, _ := json.Marshal(msg3)
			rabbitmq.PublishToExchange(l.ch, "leilao_events", queueName, body3, "text/plain")

			a.Ativo = false
			event := map[string]string{
				"id":     a.ID,
				"status": "encerrado",
			}
			body, _ := json.Marshal(event)
			// Publica na exchange com routing key
			rabbitmq.PublishToExchange(l.ch, "leilao_events", "leilao.finalizado", body)

			log.Printf("Leilão %s finalizado!", a.ID)
		}(auction)
	}
	fmt.Println("Agendamento de leilões iniciado.")
}
