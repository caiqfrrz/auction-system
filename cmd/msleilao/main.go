package main

import (
	"auction-system/internal/msleilao"
	"auction-system/pkg/rabbitmq"
	"os"
	"os/signal"
)

func main() {
	conn, ch := rabbitmq.Connect()
	defer conn.Close()
	defer ch.Close()

	msLeilao := msleilao.NewMsLeilao(ch)
	msLeilao.Start()

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, os.Interrupt)
	<-forever
}
