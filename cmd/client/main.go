package main

import (
	"auction-system/internal/client"
	"auction-system/pkg/rabbitmq"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	userID := flag.String("user", "", "User ID for the client")
	flag.Parse()

	if *userID == "" {
		fmt.Println("Usage: go run main.go -user <userID>")
		fmt.Println("Example: go run main.go -user alice")
		os.Exit(1)
	}

	conn, ch := rabbitmq.Connect()
	defer conn.Close()
	defer ch.Close()

	c := client.NewClient(ch, *userID)

	go c.ListenAuctions()

	go c.Menu()

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, os.Interrupt)
	<-forever
}
