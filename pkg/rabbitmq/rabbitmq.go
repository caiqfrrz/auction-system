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
		true,
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

func DeclareTempQueue(ch *amqp.Channel) amqp.Queue {
	q, err := ch.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Error declaring temp queue: %v", err)
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

func DeclareExchange(ch *amqp.Channel, name string, typ string) {
	err := ch.ExchangeDeclare(
		name,
		typ,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Error creating exchange %s: %v", name, err)
	}
}

func BindQueueToExchange(ch *amqp.Channel, name string, key string, exchange string) {
	err := ch.QueueBind(name, key, exchange, false, nil)
	if err != nil {
		log.Printf("Error binding queue to exchange %s: %v", name, err)
	}
}

func PublishToExchange(ch *amqp.Channel, exchange string, key string, body []byte) {
	err := ch.Publish(
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Error publishing to exchange %s: %v", exchange, err)
	}
}
