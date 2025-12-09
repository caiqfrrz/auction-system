package msleilao

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Auction struct {
	ID        string    `json:"id"`
	Descricao string    `json:"description"`
	Inicio    time.Time `json:"start"`
	Fim       time.Time `json:"end"`
	Ativo     bool      `json:"active"`
}

type MsLeilao struct {
	auctions []Auction
	mu       sync.RWMutex
	// Callbacks para notificar quando leilão inicia/finaliza
	onAuctionStarted  func(string, int64) // (leilaoID, duracao)
	onAuctionFinished func(string)        // (leilaoID)
}

func NewMsLeilao() *MsLeilao {
	return &MsLeilao{
		auctions: []Auction{},
	}
}

// Registrar callbacks para notificações
func (l *MsLeilao) SetAuctionCallbacks(onStart func(string, int64), onFinish func(string)) {
	l.onAuctionStarted = onStart
	l.onAuctionFinished = onFinish
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

func (l *MsLeilao) ScheduleAuction(auction *Auction) {
	// Goroutine para iniciar leilão
	go func(a *Auction) {
		time.Sleep(time.Until(a.Inicio))

		l.mu.Lock()
		a.Ativo = true
		l.mu.Unlock()

		log.Printf("Leilão %s iniciado!", a.ID)

		// Notificar via callback (que chamará gRPC)
		if l.onAuctionStarted != nil {
			duracao := int64(a.Fim.Sub(a.Inicio).Seconds())
			l.onAuctionStarted(a.ID, duracao)
		}
	}(auction)

	// Goroutine para finalizar leilão
	go func(a *Auction) {
		time.Sleep(time.Until(a.Fim))

		l.mu.Lock()
		a.Ativo = false
		l.mu.Unlock()

		log.Printf("Leilão %s finalizado!", a.ID)

		// Notificar via callback (que chamará gRPC)
		if l.onAuctionFinished != nil {
			l.onAuctionFinished(a.ID)
		}
	}(auction)
}
