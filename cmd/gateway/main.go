package main

import (
	"auction-system/cmd/gateway/server"
	"fmt"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("cmd/gateway/.env"); err != nil {
		fmt.Println("Warning: .env file not found, using defaults")
	}
	server := server.NewServer()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}
}
