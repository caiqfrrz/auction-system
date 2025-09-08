package rabbitmq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Connect() (*amqp.Connection, *amqp.Channel) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Error opening connection to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error opening channel: %v", err)
	}

	return conn, ch
}

func DeclareQueue(ch *amqp.Channel, name string) amqp.Queue {
	q, err := ch.QueueDeclare(
		name,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error declaring queue %s: %v", name, err)
	}
	return q
}

func Publish(ch *amqp.Channel, queue string, body []byte) {
	err := ch.Publish(
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Error publishing message to %s: %v", queue, err)
	}
}
