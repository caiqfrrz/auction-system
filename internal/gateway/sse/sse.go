package sse

import (
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type EventType string

const (
	LanceValidado   EventType = "lance_validado"
	LanceInvalidado EventType = "lance_invalidado"
	LeilaoVencedor  EventType = "leilao_vencedor"
	LinkPagamento   EventType = "link_pagamento"
	StatusPagamento EventType = "status_pagamento"
)

type Notification struct {
	Type      EventType   `json:"type"`
	LeilaoID  int         `json:"leilao_id"`
	ClienteID string      `json:"cliente_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type Client struct {
	ID       string
	LeilaoID int
	Channel  chan Notification
}

type EventStream struct {
	Message       chan Notification
	NewClients    chan Client
	ClosedClients chan Client

	ClientsByLeilao map[int]map[string]chan Notification
	ClientsByID     map[string]chan Notification
}

func (s *EventStream) listen() {
	for {
		select {
		case client := <-s.NewClients:
			s.ClientsByID[client.ID] = client.Channel

			if s.ClientsByLeilao[client.LeilaoID] == nil {
				s.ClientsByLeilao[client.LeilaoID] = make(map[string]chan Notification)
			}
			s.ClientsByLeilao[client.LeilaoID][client.ID] = client.Channel

			log.Printf("Cliente %s registrado no leilão %d", client.ID, client.LeilaoID)

		case client := <-s.ClosedClients:
			delete(s.ClientsByID, client.ID)
			if s.ClientsByLeilao[client.LeilaoID] != nil {
				delete(s.ClientsByLeilao[client.LeilaoID], client.ID)
			}
			close(client.Channel)
			log.Printf("Cliente %s desconectado", client.ID)

		case notification := <-s.Message:
			s.broadcastNotification(notification)
		}
	}
}

func (s *EventStream) broadcastNotification(notif Notification) {
	switch notif.Type {
	case LanceValidado, LeilaoVencedor:
		if clients, ok := s.ClientsByLeilao[notif.LeilaoID]; ok {
			for _, ch := range clients {
				log.Printf("mandando msg leilao vencedor %d", notif.LeilaoID)
				ch <- notif
			}
		}

	case LanceInvalidado, LinkPagamento, StatusPagamento:
		if ch, ok := s.ClientsByID[notif.ClienteID]; ok {
			ch <- notif
		}
	}
}

func (stream *EventStream) SSEConnMiddleware() gin.HandlerFunc {
	return func(gctx *gin.Context) {
		leilaoID, err := strconv.Atoi(gctx.Param("auctionID"))
		if err != nil {
			gctx.JSON(400, gin.H{"error": "invalid auctionID"})
			gctx.Abort()
			return
		}

		clienteID := gctx.Query("clienteID")
		if clienteID == "" {
			gctx.JSON(400, gin.H{"error": "clienteID needed"})
			gctx.Abort()
			return
		}

		client := Client{
			ID:       clienteID,
			LeilaoID: leilaoID,
			Channel:  make(chan Notification),
		}

		stream.NewClients <- client
		defer func() {
			log.Printf("Fechando conexão do cliente %s no leilão %d", client.ID, client.LeilaoID)
			stream.ClosedClients <- client
		}()

		gctx.Set("client", client)
		gctx.Next()
	}
}

func NewEventStream() *EventStream {
	stream := &EventStream{
		Message:         make(chan Notification),
		NewClients:      make(chan Client),
		ClosedClients:   make(chan Client),
		ClientsByLeilao: make(map[int]map[string]chan Notification),
		ClientsByID:     make(map[string]chan Notification),
	}

	go stream.listen()
	return stream
}
