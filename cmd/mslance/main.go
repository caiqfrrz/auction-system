package main

import (
	"auction-system/cmd/mslance/server"
	"auction-system/pkg/rabbitmq"
	"fmt"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	conn, ch := rabbitmq.Connect()
	defer conn.Close()
	defer ch.Close()

	server := server.NewServer(ch)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, os.Interrupt)
	<-forever
}
