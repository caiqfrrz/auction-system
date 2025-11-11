package msleilao

import (
	"auction-system/pkg/models"
	"auction-system/pkg/rabbitmq"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Auction struct {
	ID        string    `json:"id"`
	Descricao string    `json:"description"`
	Inicio    time.Time `json:"start"`
	Fim       time.Time `json:"end"`
	Ativo     bool      `json:"active"`
}

type MsLeilao struct {
	ch       *amqp.Channel
	auctions []Auction
	mu       sync.RWMutex
}

func NewMsLeilao(ch *amqp.Channel) *MsLeilao {
	// now := time.Now()
	auctions := []Auction{
		// {ID: "1", Descricao: "Almoço no RU", Inicio: now.Add(2 * time.Second), Fim: now.Add(50 * time.Second), Ativo: false},
		// {ID: "2", Descricao: "Monalisa", Inicio: now.Add(20 * time.Second), Fim: now.Add(40 * time.Second), Ativo: false},
	}

	return &MsLeilao{ch: ch, auctions: auctions}
}

func (l *MsLeilao) CreateAuction(desc string, start time.Time, end time.Time) error {
	now := time.Now()

	if strings.TrimSpace(desc) == "" {
		return fmt.Errorf("description cannot be empty")
	}

	if start.Before(now) {
		return fmt.Errorf("start time cannot be in the past")
	}

	if end.Before(now) {
		return fmt.Errorf("end time cannot be in the past")
	}

	if end.Before(start) {
		return fmt.Errorf("end time cannot be before start time")
	}

	l.mu.Lock()

	newAuction := Auction{
		ID:        strconv.Itoa(len(l.auctions) + 1),
		Descricao: desc,
		Inicio:    start,
		Fim:       end,
		Ativo:     false,
	}

	l.auctions = append(l.auctions, newAuction)
	pAuction := &l.auctions[len(l.auctions)-1]
	l.mu.Unlock()

	l.ScheduleAuction(pAuction)

	log.Printf("Leilão criado e agendado: %s (%s) - Início: %s, Fim: %s",
		newAuction.ID, newAuction.Descricao,
		newAuction.Inicio.Format(time.RFC3339),
		newAuction.Fim.Format(time.RFC3339))

	return nil
}

func (l *MsLeilao) ConsultAuctions() []Auction {
	l.mu.RLock()
	defer l.mu.RUnlock()

	auctions := make([]Auction, len(l.auctions))
	copy(auctions, l.auctions)
	return auctions
}

func (l *MsLeilao) Start() {
	rabbitmq.DeclareExchange(l.ch, "leilao_events", "topic")

	l.mu.RLock()
	for i := range l.auctions {
		l.ScheduleAuction(&l.auctions[i])
	}
	l.mu.RUnlock()

	fmt.Println("Agendamento de leilões iniciado.")
}

func (l *MsLeilao) ScheduleAuction(auction *Auction) {
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
