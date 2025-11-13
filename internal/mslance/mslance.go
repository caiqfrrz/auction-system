package mslance

import (
	"auction-system/pkg/models"
	"auction-system/pkg/rabbitmq"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type LeilaoStatus struct {
	ID         string
	Descricao  string
	Ativo      bool
	MaiorLance float64
	Vencedor   string
}

type MSLance struct {
	ch      *amqp.Channel
	leiloes map[string]*LeilaoStatus
	mu      sync.Mutex
}

func NewMSLance(ch *amqp.Channel) *MSLance {
	return &MSLance{
		ch:      ch,
		leiloes: make(map[string]*LeilaoStatus),
	}
}

// Inicializa a exchange e faz o binding das filas
func (m *MSLance) DeclareExchangeAndQueues() {
	rabbitmq.DeclareExchange(m.ch, "leilao_events", "topic")

	rabbitmq.DeclareQueue(m.ch, "lance_realizado")
	rabbitmq.BindQueueToExchange(m.ch, "lance_realizado", "lance.realizado", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "leilao_iniciado")
	rabbitmq.BindQueueToExchange(m.ch, "leilao_iniciado", "leilao.iniciado", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "leilao_finalizado")
	rabbitmq.BindQueueToExchange(m.ch, "leilao_finalizado", "leilao.finalizado", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "lance_validado")
	rabbitmq.BindQueueToExchange(m.ch, "lance_validado", "lance.validado", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "lance_invalidado")
	rabbitmq.BindQueueToExchange(m.ch, "lance_invalidado", "lance.invalidado", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "leilao_vencedor")
	rabbitmq.BindQueueToExchange(m.ch, "leilao_vencedor", "leilao.vencedor", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "mspag_leilao_vencedor")
	rabbitmq.BindQueueToExchange(m.ch, "mspag_leilao_vencedor", "leilao.vencedor", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "cliente_registrado")
	rabbitmq.BindQueueToExchange(m.ch, "cliente_registrado", "cliente.registrado", "leilao_events")
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

		invalidado := map[string]interface{}{
			"leilao_id": bid.LeilaoID,
			"user_id":   bid.UserID,
			"valor":     bid.Valor,
			"motivo":    "Leilão não está ativo",
		}
		body, _ := json.Marshal(invalidado)
		rabbitmq.PublishToExchange(m.ch, "leilao_events", "lance.invalidado", body)

		return fmt.Errorf("leilão %s não está ativo", bid.LeilaoID)
	}

	if bid.Valor <= leilao.MaiorLance {
		log.Printf("Lance invalidado: %.2f <= %.2f (leilão %s)", bid.Valor, leilao.MaiorLance, bid.LeilaoID)

		invalidado := map[string]interface{}{
			"leilao_id": bid.LeilaoID,
			"user_id":   bid.UserID,
			"valor":     bid.Valor,
			"motivo":    fmt.Sprintf("Lance deve ser maior que %.2f", leilao.MaiorLance),
		}
		body, _ := json.Marshal(invalidado)
		rabbitmq.PublishToExchange(m.ch, "leilao_events", "lance.invalidado", body)

		return fmt.Errorf("lance deve ser maior que %.2f", leilao.MaiorLance)
	}

	leilao.MaiorLance = bid.Valor
	leilao.Vencedor = bid.UserID

	bidByte, err := json.Marshal(bid)
	if err != nil {
		return err
	}

	log.Printf("✅ Lance validado: %.2f por %s (leilão %s)", bid.Valor, bid.UserID, bid.LeilaoID)
	rabbitmq.PublishToExchange(m.ch, "leilao_events", "lance.validado", bidByte)

	return nil
}

func (m *MSLance) ListenLeilaoIniciado() {
	msgs, _ := m.ch.Consume("leilao_iniciado", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var leilao models.LeilaoIniciado
			if err := json.Unmarshal(d.Body, &leilao); err == nil {
				m.mu.Lock()
				m.leiloes[leilao.ID] = &LeilaoStatus{
					ID:         leilao.ID,
					Descricao:  leilao.Descricao,
					Ativo:      true,
					MaiorLance: 0,
					Vencedor:   "",
				}
				m.mu.Unlock()
				log.Printf("Leilão iniciado: %s (%s)", leilao.Descricao, leilao.ID)
			}
		}
	}()
}

func (m *MSLance) ListenLeilaoFinalizado() {
	msgs, _ := m.ch.Consume("leilao_finalizado", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var finalizado struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(d.Body, &finalizado); err == nil {
				m.mu.Lock()
				leilao, ok := m.leiloes[finalizado.ID]
				if ok && leilao.Ativo {
					leilao.Ativo = false
					vencedor := models.LeilaoVencedor{
						LeilaoID: leilao.ID,
						UserID:   leilao.Vencedor,
						Valor:    leilao.MaiorLance,
					}
					body, _ := json.Marshal(vencedor)
					rabbitmq.PublishToExchange(m.ch, "leilao_events", "leilao.vencedor", body)
					log.Printf("Leilão %s finalizado. Vencedor: %s (%.2f)", leilao.ID, leilao.Vencedor, leilao.MaiorLance)
				}
				m.mu.Unlock()
			}
		}
	}()
}
