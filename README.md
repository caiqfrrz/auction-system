# Auction System

A distributed, event-driven auction platform built in Go, using a microservices architecture with **RabbitMQ** for asynchronous messaging and **Server-Sent Events (SSE)** for real-time updates to the browser. Auctions are scheduled and run automatically, bids are validated in real time, and the winner is routed through a simulated payment flow (payment link generation + webhook confirmation).

A React + TypeScript frontend (Vite) is included for creating auctions, placing bids, and watching live auction activity.

## Architecture

```mermaid
flowchart LR
    FE["React Frontend<br/>(Vite, :5173)"]

    subgraph Backend
        GW["Gateway API<br/>(Gin, :PORT)"]
        LEILAO["ms-leilao<br/>auction lifecycle<br/>(:8081)"]
        LANCE["ms-lance<br/>bidding logic<br/>(:8082)"]
        PAG["ms-pagamento<br/>payment orchestration<br/>(:8084)"]
    end

    EXT["pagexterno<br/>mock external<br/>payment gateway (:8085)"]
    MQ[("RabbitMQ<br/>leilao_events exchange")]

    FE -- "REST + SSE" --> GW
    GW -- "create / consult auctions" --> LEILAO
    GW -- "place bid / highest bid" --> LANCE

    LEILAO -- "leilao.iniciado / leilao.finalizado" --> MQ
    MQ -- "leilao.iniciado / leilao.finalizado" --> LANCE
    LANCE -- "lance.validado / lance.invalidado / leilao.vencedor" --> MQ
    MQ -- "leilao.vencedor" --> PAG
    PAG -- "POST /payment" --> EXT
    EXT -- "payment webhook" --> PAG
    PAG -- "link.pagamento / status.pagamento" --> MQ
    MQ -- "all client-facing events" --> GW
    GW -- "Server-Sent Events" --> FE
```

Each auction service is independent, communicates only through RabbitMQ (never calls another microservice directly), and can be scaled or restarted on its own. The Gateway is the only service the frontend talks to — it fans requests out to `ms-leilao`/`ms-lance` over REST and turns the RabbitMQ event stream into SSE for the browser.

## Services

| Service | Path | Port | Responsibility |
|---|---|---|---|
| **Gateway** | `cmd/gateway` | `PORT` env var | Public REST API + SSE endpoint; proxies requests to `ms-leilao`/`ms-lance`; consumes `leilao_events` and pushes notifications to connected clients |
| **ms-leilao** | `cmd/msleilao` | `8081` | Owns auction lifecycle: creation, validation, in-memory storage, and automatic start/end scheduling |
| **ms-lance** | `cmd/mslance` | `8082` | Tracks each active auction's highest bid, validates incoming bids, publishes bid outcomes and the final winner |
| **ms-pagamento** | `cmd/mspagamento` | `8084` (default) | Listens for auction winners, requests a payment link from the external payment service, and relays payment status back into the event bus |
| **pagexterno** | `internal/pagexterno` | `8085` | Standalone mock of a third-party payment processor (serves a simple HTML "pay/cancel" page and fires a webhook back to `ms-pagamento`) |
| **Frontend** | `frontend/` | `5173` (Vite dev) | React/TypeScript UI for creating auctions, bidding, and watching live updates |

## Event flow (RabbitMQ)

All backend services communicate through a single **topic exchange**, `leilao_events`, declared with durable queues bound to it:

| Routing key | Published by | Consumed by | Meaning |
|---|---|---|---|
| `leilao.iniciado` | ms-leilao | ms-lance, Gateway | An auction has started |
| `leilao.finalizado` | ms-leilao | ms-lance | An auction's time window has ended |
| `lance.validado` | ms-lance | Gateway | A bid was accepted as the new highest bid |
| `lance.invalidado` | ms-lance | Gateway | A bid was rejected (auction inactive, or value too low) |
| `leilao.vencedor` | ms-lance | ms-pagamento, Gateway | The auction closed; announces the winning bidder and amount |
| `link.pagamento` | ms-pagamento | Gateway | A payment link was generated for the winner |
| `status.pagamento` | ms-pagamento | Gateway | The external payment was approved or rejected |

The Gateway re-broadcasts these as Server-Sent Events: auction-wide events (`lance_validado`, `leilao_vencedor`) go to every client watching that auction, while user-specific events (`lance_invalidado`, `link_pagamento`, `status_pagamento`) go only to the relevant client.

## API reference (Gateway)

The frontend and any external client should only talk to the **Gateway**.

| Method | Path | Description |
|---|---|---|
| `GET` | `/consult-auctions` | List all auctions |
| `POST` | `/create-auction` | Create an auction — body: `{ "description": string, "start": RFC3339, "end": RFC3339 }` |
| `POST` | `/make-bid` | Place a bid — body: `{ "user_id": string, "leilao_id": string, "valor": string }` |
| `GET` | `/highest-bid?auctionId=` | Get the current highest bid for an auction |
| `GET` | `/register-interest/:auctionID/stream?clienteID=` | Open an SSE stream for real-time updates on that auction |
| `GET` | `/cancel-interest` | *(not implemented yet — returns `501`)* |

## Getting started

### Prerequisites

- Go 1.24+
- Node.js 18+ (for the frontend)
- RabbitMQ running locally (default credentials `guest:guest` on `localhost:5672`)

```bash
# quickest way to get RabbitMQ running locally
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

### Backend

```bash
git clone https://github.com/gUI-pe/auction-system.git
cd auction-system
go mod download
```

Create `cmd/gateway/.env` from the example and fill in the ports you're running (see [Configuration](#configuration)):

```bash
cp cmd/gateway/.env.example cmd/gateway/.env
```

Start the core services (auction lifecycle, bidding, and gateway) with the provided script:

```bash
./exec.sh
```

This runs `ms-leilao`, `ms-lance`, and the Gateway. The payment flow is separate and needs to be started manually if you want to exercise it end-to-end:

```bash
go run internal/pagexterno/pagexterno.go   # mock external payment processor, :8085
go run cmd/mspagamento/main.go             # payment orchestrator, :8084
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The dev server runs on `http://localhost:5173` (the Gateway's CORS policy is hardcoded to allow this origin).

## Configuration

The Gateway is the only service configured via environment variables (loaded from `cmd/gateway/.env`):

| Variable | Purpose |
|---|---|
| `PORT` | Port the Gateway listens on |
| `MSLEILAO_HOST` | Host:port of `ms-leilao` (defaults to matching its hardcoded port, `8081`) |
| `MSLANCE_HOST` | Host:port of `ms-lance` (defaults to matching its hardcoded port, `8082`) |
| `RABBITMQ_URL` | Full AMQP connection string, e.g. `amqp://guest:guest@localhost:5672/` |

> **Note:** the checked-in `cmd/gateway/.env.example` currently has values that don't line up with the services' hardcoded ports (e.g. it points `MSLEILAO_HOST` at `8080` and reuses `8082` for both `PORT` and `ms-lance`, and uses the key `RABBITMQ_HOST` instead of `RABBITMQ_URL`). Double-check these against the table above before running the Gateway, since `ms-leilao` and `ms-lance` themselves listen on hardcoded ports `8081`/`8082` regardless of what's in `.env`.

`ms-pagamento` also reads optional environment variables (all have working defaults): `EXTERNAL_PAY_URL` (default `http://localhost:8085`), `PUBLIC_URL` (default `http://localhost:8084`), and `HTTP_ADDR` (default `:8084`).

## Project layout

```
.
├── cmd/
│   ├── gateway/       # Public API + SSE gateway
│   ├── mslance/       # Bidding microservice
│   ├── msleilao/      # Auction lifecycle microservice
│   └── mspagamento/   # Payment orchestration microservice
├── internal/
│   ├── gateway/       # Gateway's RabbitMQ consumer + SSE event stream
│   ├── mslance/       # Bidding domain logic
│   ├── msleilao/      # Auction domain logic + scheduling
│   ├── mspagamento/   # Payment domain logic
│   └── pagexterno/    # Standalone mock external payment gateway
├── pkg/
│   ├── models/        # Shared event/message structs
│   └── rabbitmq/      # RabbitMQ connection + publish/consume helpers
├── frontend/          # React + TypeScript + Vite client
├── exec.sh            # Convenience script to run leilao + lance + gateway together
├── go.mod / go.sum
└── README.md
```

## Tech stack

- **Backend:** Go, [Gin](https://github.com/gin-gonic/gin), [RabbitMQ](https://www.rabbitmq.com/) (`amqp091-go`), Server-Sent Events
- **Frontend:** React 19, TypeScript, Vite, Tailwind CSS, `react-hot-toast`

## Known gaps

- `GET /cancel-interest` is a stub and returns `501 Not Implemented`.
- The payment path (`ms-pagamento` + `pagexterno`) isn't started by `exec.sh` and must be run separately.
- CORS on both the Gateway and `ms-leilao`/`ms-lance` is hardcoded to `http://localhost:5173`.
- See the [Configuration](#configuration) note above about `cmd/gateway/.env.example` needing updated values.
