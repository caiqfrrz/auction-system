package msnotis

import (
	"auction-system/pkg/rabbitmq"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MSNotis struct {
	ch *amqp.Channel
}

func NewMSNotis(ch *amqp.Channel) *MSNotis {
	return &MSNotis{ch: ch}
}

func (m *MSNotis) DeclareExchangeAndQueues() {
	rabbitmq.DeclareExchange(m.ch, "leilao_events", "topic")

	rabbitmq.DeclareQueue(m.ch, "lance_validado")
	rabbitmq.BindQueueToExchange(m.ch, "lance_validado", "lance.validado", "leilao_events")

	rabbitmq.DeclareQueue(m.ch, "leilao_vencedor")
	rabbitmq.BindQueueToExchange(m.ch, "leilao_vencedor", "leilao.vencedor", "leilao_events")
}

func (m *MSNotis) ListenAndPublish() {
	msgsLance, _ := m.ch.Consume("lance_validado", "", true, false, false, false, nil)
	msgsVencedor, _ := m.ch.Consume("leilao_vencedor", "", true, false, false, false, nil)

	go func() {
		for d := range msgsLance {
			var msg struct {
				LeilaoID string  `json:"leilao_id"`
				UserID   string  `json:"user_id"`
				Valor    float64 `json:"valor"`
			}
			if err := json.Unmarshal(d.Body, &msg); err == nil && msg.LeilaoID != "" {
				queueName := fmt.Sprintf("leilao_%s", msg.LeilaoID)
				rabbitmq.DeclareQueue(m.ch, queueName)
				rabbitmq.BindQueueToExchange(m.ch, queueName, queueName, "leilao_events")

				humanMessage := fmt.Sprintf("💰 Novo lance válido de R$ %.2f pelo usuário %s", msg.Valor, msg.UserID)
				err := m.ch.Publish(
					"leilao_events",
					queueName,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        []byte(humanMessage),
					},
				)
				if err != nil {
					log.Printf("Error publishing to leilao events: %v", err)
				}
				log.Printf("Notificação de lance_validado publicada para %s", queueName)
			} else {
				log.Printf("error msnotis %v", err)
			}
		}
	}()

	go func() {
		for d := range msgsVencedor {
			var msg struct {
				LeilaoID string  `json:"leilao_id"`
				UserID   string  `json:"user_id"`
				Valor    float64 `json:"valor"`
			}
			if err := json.Unmarshal(d.Body, &msg); err == nil && msg.LeilaoID != "" {
				queueName := fmt.Sprintf("leilao_%s", msg.LeilaoID)
				rabbitmq.DeclareQueue(m.ch, queueName)
				rabbitmq.BindQueueToExchange(m.ch, queueName, queueName, "leilao_events")

				humanMessage := fmt.Sprintf("🏆 Leilão %s finalizado! Vencedor: %s com lance de R$ %.2f", msg.LeilaoID, msg.UserID, msg.Valor)
				err := m.ch.Publish(
					"leilao_events",
					queueName,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        []byte(humanMessage),
					},
				)
				if err != nil {
					log.Printf("Error publishing to leilao events: %v", err)
				}
				log.Printf("Notificação de leilao_vencedor publicada para %s", queueName)
			}
		}
	}()
}
