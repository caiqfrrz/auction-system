package main

import (
	"auction-system/cmd/gateway/server"
	"fmt"
	"net/http"
)

func main() {
	server := server.NewServer()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}
}
