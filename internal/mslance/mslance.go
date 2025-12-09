package mslance

import (
	"auction-system/pkg/models"
	"fmt"
	"log"
	"sync"
)

type LeilaoStatus struct {
	ID         string
	Descricao  string
	Ativo      bool
	MaiorLance float64
	Vencedor   string
}

type MSLance struct {
	leiloes map[string]*LeilaoStatus
	mu      sync.Mutex
	// Callbacks para notificar eventos
	onBidValidated   func(models.LanceRealizado)
	onBidInvalidated func(string, string, float64, string) // leilaoID, userID, valor, motivo
	onAuctionWinner  func(models.LeilaoVencedor)
}

func NewMSLance() *MSLance {
	return &MSLance{
		leiloes: make(map[string]*LeilaoStatus),
	}
}

func (m *MSLance) SetBidCallbacks(
	onValidated func(models.LanceRealizado),
	onInvalidated func(string, string, float64, string),
	onWinner func(models.LeilaoVencedor),
) {
	m.onBidValidated = onValidated
	m.onBidInvalidated = onInvalidated
	m.onAuctionWinner = onWinner
}

func (m *MSLance) MakeBid(bid models.LanceRealizado) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	leilao, ok := m.leiloes[bid.LeilaoID]
	if !ok {
		log.Printf("Leilão %s não encontrado", bid.LeilaoID)
		return fmt.Errorf("leilão %s não encontrado", bid.LeilaoID)
	}

	if !leilao.Ativo {
		log.Printf("Leilão %s não está ativo", bid.LeilaoID)

		// Notificar via callback
		if m.onBidInvalidated != nil {
			m.onBidInvalidated(bid.LeilaoID, bid.UserID, bid.Valor, "Leilão não está ativo")
		}

		return fmt.Errorf("leilão %s não está ativo", bid.LeilaoID)
	}

	if bid.Valor <= leilao.MaiorLance {
		log.Printf("Lance invalidado: %.2f <= %.2f (leilão %s)", bid.Valor, leilao.MaiorLance, bid.LeilaoID)

		motivo := fmt.Sprintf("Lance deve ser maior que %.2f", leilao.MaiorLance)

		// Notificar via callback
		if m.onBidInvalidated != nil {
			m.onBidInvalidated(bid.LeilaoID, bid.UserID, bid.Valor, motivo)
		}

		return fmt.Errorf(motivo)
	}

	leilao.MaiorLance = bid.Valor
	leilao.Vencedor = bid.UserID

	log.Printf("✅ Lance validado: %.2f por %s (leilão %s)", bid.Valor, bid.UserID, bid.LeilaoID)

	if m.onBidValidated != nil {
		m.onBidValidated(bid)
	}

	return nil
}

func (m *MSLance) GetHighestBid(auctionID string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	auction, ok := m.leiloes[auctionID]
	if !ok {
		log.Printf("Leilão %s não encontrado", auctionID)
		return 0, fmt.Errorf("leilão %s não encontrado", auctionID)
	}

	return auction.MaiorLance, nil
}

// Chamado via gRPC
func (m *MSLance) HandleAuctionStarted(leilaoID string, duracao int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.leiloes[leilaoID] = &LeilaoStatus{
		ID:         leilaoID,
		Ativo:      true,
		MaiorLance: 0,
		Vencedor:   "",
	}

	log.Printf("Leilão iniciado: %s", leilaoID)
}

// Chamado via gRPC
func (m *MSLance) HandleAuctionFinished(leilaoID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	leilao, ok := m.leiloes[leilaoID]
	if ok && leilao.Ativo {
		leilao.Ativo = false

		if leilao.Vencedor != "" {
			vencedor := models.LeilaoVencedor{
				LeilaoID: leilao.ID,
				UserID:   leilao.Vencedor,
				Valor:    leilao.MaiorLance,
			}

			log.Printf("Leilão %s finalizado. Vencedor: %s (%.2f)",
				leilao.ID, leilao.Vencedor, leilao.MaiorLance)

			if m.onAuctionWinner != nil {
				m.onAuctionWinner(vencedor)
			}
		}
	}
}
