package client

import (
	"auction-system/pkg/models"
	"auction-system/pkg/rabbitmq"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jroimartin/gocui"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	ch        *amqp.Channel
	userID    string
	priv      *rsa.PrivateKey
	pub       *rsa.PublicKey
	gui       *gocui.Gui
	listening map[string]bool // leilaoID -> true se já está ouvindo
}

func NewClient(ch *amqp.Channel, userID string) *Client {
	priv, pub := generateKeys()
	return &Client{ch: ch, userID: userID, priv: priv, pub: pub, listening: make(map[string]bool)}
}

func generateKeys() (*rsa.PrivateKey, *rsa.PublicKey) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	return priv, &priv.PublicKey
}

func signMessage(priv *rsa.PrivateKey, message []byte) string {
	hashed := sha256.Sum256(message)
	signature, _ := rsa.SignPKCS1v15(rand.Reader, priv, 0, hashed[:])
	return base64.StdEncoding.EncodeToString(signature)
}

func (c *Client) ListenAuctions() {
	// Declara exchange (idempotente)
	rabbitmq.DeclareExchange(c.ch, "leilao_events", "topic")
	q := rabbitmq.DeclareTempQueue(c.ch)

	// Faz o binding para receber todos os leilões iniciados
	rabbitmq.BindQueueToExchange(c.ch, q.Name, "leilao.iniciado", "leilao_events")

	msgs, _ := c.ch.Consume(q.Name, "", true, false, false, false, nil)

	for d := range msgs {
		var auction models.LeilaoIniciado
		if err := json.Unmarshal(d.Body, &auction); err == nil {
			c.gui.Update(func(g *gocui.Gui) error {
				v, _ := g.View("auctions")
				fmt.Fprintf(v, "Novo leilão, id: %s | %s\n", auction.ID, auction.Descricao)
				return nil
			})
		}
	}
}

func (c *Client) SendBid(auctionID string, value float64) {
	bid := models.LanceRealizado{
		LeilaoID:   auctionID,
		UserID:     c.userID,
		Valor:      value,
		Assinatura: "",
	}

	body, _ := json.Marshal(bid)

	bid.Assinatura = signMessage(c.priv, body)

	bodyWithSignature, _ := json.Marshal(bid)

	rabbitmq.PublishToExchange(c.ch, "leilao_events", "lance.realizado", bodyWithSignature)

	c.ListenNotifications(auctionID)

	c.gui.Update(func(g *gocui.Gui) error {
		v, _ := g.View("notifications")
		fmt.Fprintf(v, "[Leilão %s] Você tentou por um lance: %.2f\n", auctionID, value)
		return nil
	})
}

func (c *Client) RegisterPublicKey() {
	// Converte chave pública para PEM
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(c.pub)
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	registro := models.ClienteRegistrado{
		UserID:    c.userID,
		PublicKey: string(pubKeyPEM),
	}
	body, _ := json.Marshal(registro)
	rabbitmq.PublishToExchange(c.ch, "leilao_events", "cliente.registrado", body)
}

func (c *Client) ListenNotifications(auctionID string) {
	if c.listening[auctionID] {
		return // já está ouvindo esse leilão
	}
	c.listening[auctionID] = true

	// Cria fila exclusiva para notificações desse leilão
	q := rabbitmq.DeclareTempQueue(c.ch)

	// Faz o binding para a fila leilao_{id}
	queueName := fmt.Sprintf("leilao_%s", auctionID)
	rabbitmq.BindQueueToExchange(c.ch, q.Name, queueName, "leilao_events")

	msgs, _ := c.ch.Consume(q.Name, "", true, false, false, false, nil)
	go func() {
		for d := range msgs {
			c.gui.Update(func(g *gocui.Gui) error {
				v, _ := g.View("notifications")
				fmt.Fprintf(v, "[Leilão %s] %s\n", auctionID, string(d.Body))
				return nil
			})
		}
	}()
}

func (c *Client) handleEnter(g *gocui.Gui, v *gocui.View) error {
	line := strings.TrimSpace(v.Buffer())
	v.Clear()
	v.SetCursor(0, 0)

	parts := strings.Split(line, " ")
	if len(parts) == 2 {
		auctionID := parts[0]
		value, err := strconv.ParseFloat(parts[1], 64)
		if err == nil {
			c.SendBid(auctionID, value)
		}
	} else if parts[0] == "q" {
		return gocui.ErrQuit
	}
	return nil
}

func (c *Client) Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("auctions", 0, 0, maxX/2-1, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Novos leilões!!"
		v.Wrap = true
	}

	if v, err := g.SetView("notifications", maxX/2, 0, maxX-1, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "🔔 Leilões inscritos!"
		v.Wrap = true
	}

	if v, err := g.SetView("input", 0, maxY-4, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Para fazer lances: <leilão-id> <valor> (ENTER to bid)"
		v.Editable = true
		if _, err := g.SetCurrentView("input"); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Menu() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalf("Error initializing gocui: %v", err)
	}
	defer g.Close()
	c.gui = g

	g.SetManagerFunc(c.Layout)

	if err := g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, c.handleEnter); err != nil {
		log.Fatal(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalf("Error in gocui main loop: %v", err)
	}
}
