package mslance

import (
	"auction-system/pkg/models"
	"auction-system/pkg/rabbitmq"
	"encoding/json"
	"log"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Simulação de chaves públicas dos clientes
var publicKeys = map[string]string{
	"user1": "publickey1"}

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

var leiloes = make(map[string]*LeilaoStatus)
var leiloesMu sync.Mutex

// Inicializa as filas necessárias
func (m *MSLance) DeclareQueues() {
	rabbitmq.DeclareQueue(m.ch, "lance_realizado")
	rabbitmq.DeclareQueue(m.ch, "leilao_iniciado")
	rabbitmq.DeclareQueue(m.ch, "leilao_finalizado")
	rabbitmq.DeclareQueue(m.ch, "lance_validado")
	rabbitmq.DeclareQueue(m.ch, "leilao_vencedor")
}

func (m *MSLance) ListenLeilaoIniciado() {
	msgs, _ := m.ch.Consume("leilao_iniciado", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var leilao models.LeilaoIniciado
			if err := json.Unmarshal(d.Body, &leilao); err == nil {
				m.mu.Lock()
				m.leiloes[leilao.ID] = &LeilaoStatus{
					ID:        leilao.ID,
					Descricao: leilao.Descricao,
					Ativo:     true,
				}
				m.mu.Unlock()
				log.Printf("Leilão iniciado: %s (%s)", leilao.Descricao, leilao.ID)
			}
		}
	}()
}

func (m *MSLance) ListenLanceRealizado() {
	msgs, _ := m.ch.Consume("lance_realizado", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var lance models.LanceRealizado
			if err := json.Unmarshal(d.Body, &lance); err != nil {
				log.Printf("Lance inválido (json): %v", err)
				continue
			}

			if !verificaAssinatura(lance) {
				log.Printf("Assinatura inválida para lance de %s", lance.UserID)
				continue
			}

			m.mu.Lock()
			leilao, ok := m.leiloes[lance.LeilaoID]
			if !ok || !leilao.Ativo {
				m.mu.Unlock()
				log.Printf("Leilão %s não existe ou não está ativo", lance.LeilaoID)
				continue
			}
			if lance.Valor > leilao.MaiorLance {
				leilao.MaiorLance = lance.Valor
				leilao.Vencedor = lance.UserID
				m.mu.Unlock()

				validado := models.LanceValidado{
					LeilaoID: lance.LeilaoID,
					UserID:   lance.UserID,
					Valor:    lance.Valor,
				}
				body, _ := json.Marshal(validado)
				rabbitmq.Publish(m.ch, "lance_validado", body)
				log.Printf("Lance validado: %+v", validado)
			} else {
				m.mu.Unlock()
				log.Printf("Lance menor que o atual para leilão %s", lance.LeilaoID)
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
					rabbitmq.Publish(m.ch, "leilao_vencedor", body)
					log.Printf("Leilão %s finalizado. Vencedor: %s (%.2f)", leilao.ID, leilao.Vencedor, leilao.MaiorLance)
				}
				m.mu.Unlock()
			}
		}
	}()
}

// Simulação de verificação de assinatura digital
func verificaAssinatura(lance models.LanceRealizado) bool {
	// Aqui você implementaria a verificação real usando a chave pública do usuário
	// Por enquanto, retorna true para simular
	return true
}
