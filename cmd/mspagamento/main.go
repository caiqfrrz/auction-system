package main

import (
	"auction-system/internal/mspagamento"
	"auction-system/pkg/rabbitmq"
	"fmt"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	// Connect to RabbitMQ
	conn, ch := rabbitmq.Connect()
	defer conn.Close()
	defer ch.Close()

	// Environment variables or defaults
	externalPayURL := os.Getenv("EXTERNAL_PAY_URL")
	if externalPayURL == "" {
		externalPayURL = "http://localhost:8085" // PagExterno service
	}
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:8084" // used for webhook callback
	}
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8084"
	}

	// Create MS Pagamento instance
	ms := mspagamento.NewMsPagamento(ch, externalPayURL, publicURL, "ms_pagamentos", httpAddr)

	// Start background listeners
	go ms.Start()

	fmt.Println("[MS PAGAMENTO] Running at", httpAddr)

	// Graceful shutdown (Ctrl+C)
	forever := make(chan os.Signal, 1)
	signal.Notify(forever, os.Interrupt)
	<-forever

	fmt.Println("\n[MS PAGAMENTO] Shutting down gracefully...")
	if err := http.DefaultClient.CloseIdleConnections; err != nil {
		fmt.Println("error closing HTTP connections:", err)
	}
}