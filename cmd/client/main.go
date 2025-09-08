package main

import (
	"auction-system/internal/client"
	"auction-system/pkg/rabbitmq"
	"os"
	"os/signal"
)

func main() {
	conn, ch := rabbitmq.Connect()
	defer conn.Close()
	defer ch.Close()

	c := client.NewClient(ch, "user")

	go c.ListenAuctions()

	go c.Menu()

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, os.Interrupt)
	<-forever
}
