package mslance

import (
	"auction-system/pkg/models"

	"crypto/rsa"

	"crypto/x509"

	"encoding/json"
	"encoding/pem"
	"log"
	"sync"

	"auction-system/pkg/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

var publicKeys = make(map[string]*rsa.PublicKey)

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

	rabbitmq.DeclareQueue(m.ch, "leilao_vencedor")
	rabbitmq.BindQueueToExchange(m.ch, "leilao_vencedor", "leilao.vencedor", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "cliente_registrado")
	rabbitmq.BindQueueToExchange(m.ch, "cliente_registrado", "cliente.registrado", "leilao_events")
}

func (m *MSLance) MakeBid(bid models.LanceRealizado) error {
	bidByte, err := json.Marshal(bid)
	if err != nil {
		return err
	}

	rabbitmq.PublishToExchange(m.ch, "leilao_events", "lance.realizado", bidByte)
	return nil
}

func (m *MSLance) ListenClienteRegistrado() {
	msgs, _ := m.ch.Consume("cliente_registrado", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var cliente models.ClienteRegistrado
			if err := json.Unmarshal(d.Body, &cliente); err == nil {
				block, _ := pem.Decode([]byte(cliente.PublicKey))
				if block != nil {
					pub, err := x509.ParsePKIXPublicKey(block.Bytes)
					if err == nil {
						publicKeys[cliente.UserID] = pub.(*rsa.PublicKey)
						log.Printf("Chave pública registrada para usuário: %s", cliente.UserID)
					} else {
						log.Printf("Erro ao parsear chave pública do usuário %s: %v", cliente.UserID, err)
					}
				}
			} else {
				log.Printf("Erro ao deserializar registro de cliente: %v", err)
			}
		}
	}()
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

func (m *MSLance) ListenLanceRealizado() {
	msgs, _ := m.ch.Consume("lance_realizado", "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			var lance models.LanceRealizado
			if err := json.Unmarshal(d.Body, &lance); err != nil {
				log.Printf("Lance inválido (json): %v", err)
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
				m.ch.Publish(
					"leilao_events",
					"lance.validado",
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					},
				)
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
					m.ch.Publish(
						"leilao_events",
						"leilao.vencedor",
						false,
						false,
						amqp.Publishing{
							ContentType: "application/json",
							Body:        body,
						},
					)
					log.Printf("Leilão %s finalizado. Vencedor: %s (%.2f)", leilao.ID, leilao.Vencedor, leilao.MaiorLance)
				}
				m.mu.Unlock()
			}
		}
	}()
}
