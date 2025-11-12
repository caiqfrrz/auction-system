package rabbitmq

import (
	"auction-system/internal/gateway/sse"
	"auction-system/pkg/models"
	"encoding/json"
	"log"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	eventStream *sse.EventStream
}

func NewRabbitMQConsumer(rabbitURL string, eventStream *sse.EventStream) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RabbitMQConsumer{
		conn:        conn,
		channel:     ch,
		eventStream: eventStream,
	}, nil
}

func (r *RabbitMQConsumer) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

func (r *RabbitMQConsumer) setupQueues() error {
	err := r.channel.ExchangeDeclare(
		"leilao_events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	queuesBindings := map[string]string{
		// "link_pagamento":   "link.pagamento",
		// "status_pagamento": "status.pagamento",
		"lance_validado":   "lance.validado",
		"lance_invalidado": "lance.invalidado",
		"leilao_vencedor":  "leilao.vencedor",
	}

	for queueName, routingKey := range queuesBindings {
		_, err := r.channel.QueueDeclare(
			queueName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return err
		}

		err = r.channel.QueueBind(
			queueName,
			routingKey,
			"leilao_events",
			false,
			nil,
		)
		if err != nil {
			return err
		}

		log.Printf("Queue %s bound to exchange with routing key %s", queueName, routingKey)
	}

	return nil
}

func (r *RabbitMQConsumer) ConsumeQueues() error {
	if err := r.setupQueues(); err != nil {
		return err
	}

	queues := map[string]func(amqp.Delivery){
		"lance_validado":   r.handleLanceValidado,
		"leilao_vencedor":  r.handleLeilaoVencedor,
		"lance_invalidado": r.handleLanceInvalidado,
	}

	for queueName, handler := range queues {
		if err := r.consumeQueue(queueName, handler); err != nil {
			return err
		}
	}

	return nil
}

func (r *RabbitMQConsumer) consumeQueue(queueName string, handler func(amqp.Delivery)) error {
	err := r.channel.Qos(1, 0, false)
	if err != nil {
		return err
	}

	msgs, err := r.channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			handler(msg)
		}
	}()

	log.Printf("Started consuming queue: %s", queueName)
	return nil
}

// func (r *RabbitMQConsumer) handleStatusPagamento(msg amqp.Delivery) {
// 	var lance models.
// 	if err := json.Unmarshal(msg.Body, &lance); err != nil {
// 		log.Printf("Error parsing lance_realizado: %v", err)
// 		msg.Nack(false, false)
// 		return
// 	}

// 	log.Printf("Lance validado: user=%s, leilao=%s, valor=%.2f", lance.UserID, lance.LeilaoID, lance.Valor)

// 	leilaoID, _ := strconv.Atoi(lance.LeilaoID)

// 	notification := sse.Notification{
// 		Type:      sse.LanceValidado,
// 		LeilaoID:  leilaoID,
// 		ClienteID: "",
// 		Data: map[string]interface{}{
// 			"user_id":   lance.UserID,
// 			"valor":     lance.Valor,
// 			"leilao_id": lance.LeilaoID,
// 		},
// 		Timestamp: time.Now(),
// 	}

// 	r.eventStream.Message <- notification
// 	msg.Ack(false)
// }

func (r *RabbitMQConsumer) handleLanceValidado(msg amqp.Delivery) {
	var lance models.LanceValidado
	if err := json.Unmarshal(msg.Body, &lance); err != nil {
		log.Printf("Error parsing lance_validado: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("Lance validado: user=%s, leilao=%s, valor=%.2f", lance.UserID, lance.LeilaoID, lance.Valor)

	leilaoID, _ := strconv.Atoi(lance.LeilaoID)

	notification := sse.Notification{
		Type:      sse.LanceValidado,
		LeilaoID:  leilaoID,
		ClienteID: "",
		Data: map[string]interface{}{
			"user_id":   lance.UserID,
			"valor":     lance.Valor,
			"leilao_id": lance.LeilaoID,
		},
		Timestamp: time.Now(),
	}

	r.eventStream.Message <- notification
	msg.Ack(false)
}

func (r *RabbitMQConsumer) handleLanceInvalidado(msg amqp.Delivery) {
	var lance models.LanceInvalidado
	if err := json.Unmarshal(msg.Body, &lance); err != nil {
		log.Printf("Error parsing lance_invalidado: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("Lance invalidado: user=%s, leilao=%s, valor=%.2f", lance.UserID, lance.LeilaoID, lance.Valor)

	leilaoID, _ := strconv.Atoi(lance.LeilaoID)

	notification := sse.Notification{
		Type:      sse.LanceInvalidado,
		LeilaoID:  leilaoID,
		ClienteID: lance.UserID,
		Data: map[string]interface{}{
			"user_id":   lance.UserID,
			"valor":     lance.Valor,
			"leilao_id": lance.LeilaoID,
			"motivo":    lance.Motivo,
		},
		Timestamp: time.Now(),
	}

	r.eventStream.Message <- notification
	msg.Ack(false)
}

func (r *RabbitMQConsumer) handleLeilaoVencedor(msg amqp.Delivery) {
	var vencedor models.LeilaoVencedor
	if err := json.Unmarshal(msg.Body, &vencedor); err != nil {
		log.Printf("Error parsing leilao_vencedor: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("Vencedor: user=%s, leilao=%s, valor=%.2f", vencedor.UserID, vencedor.LeilaoID, vencedor.Valor)

	leilaoID, _ := strconv.Atoi(vencedor.LeilaoID)

	notification := sse.Notification{
		Type:     sse.LeilaoVencedor,
		LeilaoID: leilaoID,
		Data: map[string]interface{}{
			"vencedor_id": vencedor.UserID,
			"valor_final": vencedor.Valor,
			"leilao_id":   vencedor.LeilaoID,
		},
		Timestamp: time.Now(),
	}

	r.eventStream.Message <- notification
	msg.Ack(false)
}
