package msnotis

import (
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
	m.ch.ExchangeDeclare(
		"leilao_events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)

	m.ch.QueueDeclare("lance_validado", true, false, false, false, nil)
	m.ch.QueueBind("lance_validado", "lance.validado", "leilao_events", false, nil)

	m.ch.QueueDeclare("leilao_vencedor", true, false, false, false, nil)
	m.ch.QueueBind("leilao_vencedor", "leilao.vencedor", "leilao_events", false, nil)
}

func (m *MSNotis) ListenAndPublish() {
	msgsLance, _ := m.ch.Consume("lance_validado", "", true, false, false, false, nil)
	msgsVencedor, _ := m.ch.Consume("leilao_vencedor", "", true, false, false, false, nil)

	go func() {
		for d := range msgsLance {
			var msg struct {
				LeilaoID string `json:"leilao_id"`
			}
			if err := json.Unmarshal(d.Body, &msg); err == nil && msg.LeilaoID != "" {
				queueName := fmt.Sprintf("leilao_%s", msg.LeilaoID)
				m.ch.QueueDeclare(queueName, true, false, false, false, nil)
				m.ch.QueueBind(queueName, queueName, "leilao_events", false, nil)
				m.ch.Publish(
					"leilao_events",
					queueName,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        d.Body,
					},
				)
				log.Printf("Notificação de lance_validado publicada para %s", queueName)
			}
		}
	}()

	go func() {
		for d := range msgsVencedor {
			var msg struct {
				LeilaoID string `json:"leilao_id"`
			}
			if err := json.Unmarshal(d.Body, &msg); err == nil && msg.LeilaoID != "" {
				queueName := fmt.Sprintf("leilao_%s", msg.LeilaoID)
				m.ch.QueueDeclare(queueName, true, false, false, false, nil)
				m.ch.QueueBind(queueName, queueName, "leilao_events", false, nil)
				m.ch.Publish(
					"leilao_events",
					queueName,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        d.Body,
					},
				)
				log.Printf("Notificação de leilao_vencedor publicada para %s", queueName)
			}
		}
	}()
}
