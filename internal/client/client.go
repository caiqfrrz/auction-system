package client

import (
	"auction-system/pkg/models"
	"auction-system/pkg/rabbitmq"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	ch     *amqp.Channel
	userID string
}

func NewClient(ch *amqp.Channel, userID string) *Client {
	return &Client{ch: ch, userID: userID}
}

func (c *Client) ListenAuctions() {
	q := rabbitmq.DeclareQueue(c.ch, "leilao_iniciado")
	msgs, _ := c.ch.Consume(q.Name, "", true, false, false, false, nil)

	for d := range msgs {
		var auction models.LeilaoIniciado
		if err := json.Unmarshal(d.Body, &auction); err == nil {
			fmt.Printf("\n Leilão iniciado: %s (id: %s)", auction.Descricao, auction.ID)
		}
	}
}

func (c *Client) SendBid(auctionID string, value float64) {
	bid := models.LanceRealizado{
		LeilaoID:   auctionID,
		UserID:     c.userID,
		Valor:      value,
		Assinatura: "fake", // TODO
	}

	body, _ := json.Marshal(bid)
	rabbitmq.Publish(c.ch, "lance_realizado", body)
	fmt.Printf("Lance enviado: %+v\n", bid)
}

func (c *Client) Menu() {
	for {
		fmt.Println("\n === Cliente leilão ===")
		fmt.Println("1. Dar lance")
		fmt.Println("Escolha: ")
		var option int
		fmt.Scan(&option)

		if option == 1 {
			var auctionID string
			var value float64
			fmt.Print("Digite o ID do leilão: ")
			fmt.Scan(&auctionID)
			fmt.Print("Digite o valor: ")
			fmt.Scan(&value)
			c.SendBid(auctionID, value)
		}
	}
}
