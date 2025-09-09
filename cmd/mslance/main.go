package main

import (
	"auction-system/internal/mslance"
	"auction-system/pkg/rabbitmq"
	"os"
	"os/signal"
)

func main() {
	conn, ch := rabbitmq.Connect()
	defer conn.Close()
	defer ch.Close()

	msl := mslance.NewMSLance(ch)

	msl.DeclareExchangeAndQueues()
	msl.ListenLeilaoIniciado()
	msl.ListenLanceRealizado()
	msl.ListenLeilaoFinalizado()

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, os.Interrupt)
	<-forever
}
