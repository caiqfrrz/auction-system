package main

import (
	"auction-system/internal/msnotis"
	"auction-system/pkg/rabbitmq"
	"os"
	"os/signal"
)

func main() {
	conn, ch := rabbitmq.Connect()
	defer conn.Close()
	defer ch.Close()

	notis := msnotis.NewMSNotis(ch)
	notis.DeclareExchangeAndQueues()
	notis.ListenAndPublish()

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, os.Interrupt)
	<-forever
}
