# NimsForest Prototype

## Overview

Build an event-driven organizational orchestration system in Go using NATS and JetStream.

## Architecture

### The Forest Metaphor

| Layer | Implementation | Purpose |
|-------|---------------|---------|
| River | JetStream stream | Unstructured external data flowing in |
| Tree | Pattern matcher | Parses river data, produces structured leaves |
| Leaf | Structured event | Named, typed event with schema |
| Wind | NATS Core pub/sub | Carries leaves |
| Nim | Static logic class | Business logic, catches leaves, acts |
| Humus | JetStream stream | Persistent state changes |
| Soil | JetStream KV | Current state |

### How It Flows

```
river (unstructured: webhooks, APIs, raw data)
    ↓
tree (pattern match, parse, structure)
    ↓
leaf (named event: "payment.completed")
    ↓
wind (carries leaf)
    ↓
nim (business logic: decide, act)
    ↓
leaf (wind) and/or compost (humus)
    ↓
soil (current state)
```

### Trees vs Nims

**Trees** are the edge layer:
- Watch the river for patterns
- Parse unstructured data
- Emit structured leaves
- No business logic, just transformation

**Nims** are the core:
- Static classes holding business logic
- Catch leaves from wind
- Make decisions
- Drop new leaves or compost state changes

## Tech Stack

- Go 1.22+
- github.com/nats-io/nats.go
- NATS Server with JetStream enabled

## Project Structure

```
nimsforest/
├── cmd/
│   └── forest/
│       └── main.go              # Starts the forest
├── internal/
│   ├── core/
│   │   ├── leaf.go              # Leaf type definition
│   │   ├── tree.go              # Base tree interface (pattern matcher)
│   │   ├── nim.go               # Base nim interface (business logic)
│   │   ├── wind.go              # NATS Core pub/sub wrapper
│   │   ├── river.go             # JetStream stream for external data
│   │   ├── humus.go             # JetStream stream for state changes
│   │   └── soil.go              # JetStream KV for current state
│   ├── trees/
│   │   ├── payment.go           # Example: parses Stripe webhooks
│   │   └── crm.go               # Example: parses CRM events
│   ├── nims/
│   │   ├── aftersales.go        # Example: followup logic
│   │   └── inventory.go         # Example: stock management
│   └── leaves/
│       └── types.go             # Leaf type definitions
├── docker-compose.yml
├── go.mod
└── README.md
```

## Implementation Spec

### 1. Leaf Types

```go
// internal/core/leaf.go

type Leaf struct {
    Subject   string          `json:"subject"`   // e.g. "payment.completed"
    Data      json.RawMessage `json:"data"`      // structured payload
    Source    string          `json:"source"`    // tree or nim that created it
    Timestamp time.Time       `json:"ts"`
}

// internal/leaves/types.go

type PaymentCompleted struct {
    CustomerID string  `json:"customer_id"`
    Amount     float64 `json:"amount"`
    Currency   string  `json:"currency"`
    ItemID     string  `json:"item_id"`
}

type FollowupRequired struct {
    CustomerID string    `json:"customer_id"`
    Reason     string    `json:"reason"`
    DueDate    time.Time `json:"due_date"`
}
```

### 2. Tree Interface

```go
// internal/core/tree.go

// Tree watches the river and produces structured leaves
type Tree interface {
    Name() string
    
    // Patterns returns the river patterns this tree watches
    Patterns() []string
    
    // Parse attempts to match and structure river data
    // Returns nil if pattern doesn't match
    Parse(subject string, data []byte) *Leaf
    
    Start(ctx context.Context) error
    Stop() error
}

type BaseTree struct {
    name string
    wind *Wind
}

func NewBaseTree(name string, wind *Wind) *BaseTree

// Drop sends a structured leaf onto the wind
func (t *BaseTree) Drop(leaf Leaf) error
```

### 3. Nim Interface

```go
// internal/core/nim.go

// Nim holds business logic and reacts to leaves
type Nim interface {
    Name() string
    
    // Subjects returns wind subjects this nim listens to
    Subjects() []string
    
    // Handle processes a caught leaf
    Handle(ctx context.Context, leaf Leaf) error
    
    Start(ctx context.Context) error
    Stop() error
}

type BaseNim struct {
    name  string
    wind  *Wind
    humus *Humus
    soil  *Soil
}

func NewBaseNim(name string, wind *Wind, humus *Humus, soil *Soil) *BaseNim

// Leaf drops a new leaf on the wind
func (n *BaseNim) Leaf(subject string, data []byte) error

// Compost sends a state change to humus
func (n *BaseNim) Compost(entity string, action string, data []byte) (uint64, error)

// Dig reads current state from soil
func (n *BaseNim) Dig(entity string) ([]byte, uint64, error)

// Bury writes state to soil with optimistic locking
func (n *BaseNim) Bury(entity string, data []byte, expectedRevision uint64) error
```

### 4. Wind (NATS Core)

```go
// internal/core/wind.go

type Wind struct {
    nc *nats.Conn
}

func NewWind(nc *nats.Conn) *Wind

// Drop sends a leaf on the wind
func (w *Wind) Drop(leaf Leaf) error

// Catch listens for leaves matching a subject pattern
func (w *Wind) Catch(subject string, handler func(leaf Leaf)) (*nats.Subscription, error)
```

### 5. River (JetStream Stream)

```go
// internal/core/river.go

type RiverData struct {
    Subject   string    `json:"subject"`   // raw source identifier
    Data      []byte    `json:"data"`      // unstructured payload
    Timestamp time.Time `json:"ts"`
}

type River struct {
    js     nats.JetStreamContext
    stream string
}

func NewRiver(js nats.JetStreamContext) (*River, error)

// Flow adds unstructured external data to the river
func (r *River) Flow(subject string, data []byte) error

// Observe watches for river data matching patterns
func (r *River) Observe(pattern string, handler func(data RiverData)) error
```

### 6. Humus (JetStream Stream)

```go
// internal/core/humus.go

type Compost struct {
    Entity    string          `json:"entity"`
    Action    string          `json:"action"`    // create, update, delete
    Data      json.RawMessage `json:"data"`
    NimName   string          `json:"nim"`
    Timestamp time.Time       `json:"ts"`
    Slot      uint64          `json:"slot"`
}

type Humus struct {
    js     nats.JetStreamContext
    stream string
}

func NewHumus(js nats.JetStreamContext) (*Humus, error)

// Add composts a state change
func (h *Humus) Add(nimName, entity, action string, data []byte) (uint64, error)

// Decompose processes compost entries
func (h *Humus) Decompose(handler func(compost Compost)) error
```

### 7. Soil (JetStream KV)

```go
// internal/core/soil.go

type Soil struct {
    kv nats.KeyValue
}

func NewSoil(js nats.JetStreamContext) (*Soil, error)

// Dig reads current state
func (s *Soil) Dig(entity string) ([]byte, uint64, error)

// Bury writes state with optimistic locking
func (s *Soil) Bury(entity string, data []byte, expectedRevision uint64) error

// Watch observes changes
func (s *Soil) Watch(pattern string, handler func(entity string, data []byte, revision uint64)) error
```

### 8. Example Tree

```go
// internal/trees/payment.go

// PaymentTree parses payment provider webhooks into structured leaves

type PaymentTree struct {
    *core.BaseTree
    river *core.River
}

func NewPaymentTree(base *core.BaseTree, river *core.River) *PaymentTree

func (t *PaymentTree) Name() string { return "payment" }

func (t *PaymentTree) Patterns() []string {
    return []string{"stripe.>", "paypal.>"}
}

func (t *PaymentTree) Start(ctx context.Context) error {
    // Watch for Stripe webhooks
    t.river.Observe("stripe.>", func(data core.RiverData) {
        leaf := t.parseStripe(data)
        if leaf != nil {
            t.Drop(*leaf)
        }
    })
    return nil
}

func (t *PaymentTree) parseStripe(data core.RiverData) *core.Leaf {
    var webhook map[string]interface{}
    json.Unmarshal(data.Data, &webhook)
    
    eventType, _ := webhook["type"].(string)
    
    switch eventType {
    case "charge.succeeded":
        // Extract and structure
        payment := leaves.PaymentCompleted{
            CustomerID: extractCustomer(webhook),
            Amount:     extractAmount(webhook),
            Currency:   extractCurrency(webhook),
            ItemID:     extractItem(webhook),
        }
        payloadBytes, _ := json.Marshal(payment)
        return &core.Leaf{
            Subject:   "payment.completed",
            Data:      payloadBytes,
            Source:    t.Name(),
            Timestamp: time.Now(),
        }
    case "charge.failed":
        // ... structure as payment.failed
    }
    return nil
}
```

### 9. Example Nim

```go
// internal/nims/aftersales.go

// AfterSalesNim handles post-purchase business logic

type AfterSalesNim struct {
    *core.BaseNim
}

func NewAfterSalesNim(base *core.BaseNim) *AfterSalesNim

func (n *AfterSalesNim) Name() string { return "aftersales" }

func (n *AfterSalesNim) Subjects() []string {
    return []string{"payment.completed", "payment.refunded"}
}

func (n *AfterSalesNim) Start(ctx context.Context) error {
    for _, subject := range n.Subjects() {
        n.wind.Catch(subject, func(leaf core.Leaf) {
            n.Handle(ctx, leaf)
        })
    }
    return nil
}

func (n *AfterSalesNim) Handle(ctx context.Context, leaf core.Leaf) error {
    switch leaf.Subject {
    case "payment.completed":
        var payment leaves.PaymentCompleted
        json.Unmarshal(leaf.Data, &payment)
        
        // Business logic: create followup task
        task := map[string]interface{}{
            "customer_id": payment.CustomerID,
            "type":        "followup",
            "due_date":    time.Now().Add(7 * 24 * time.Hour),
            "status":      "pending",
        }
        taskBytes, _ := json.Marshal(task)
        
        // Compost to humus (persistent)
        n.Compost("tasks/followup-"+payment.CustomerID, "create", taskBytes)
        
        // Drop thank-you leaf (ephemeral)
        n.Leaf("comms.email.send", leaf.Data)
        
    case "payment.refunded":
        // Different business logic...
    }
    return nil
}
```

### 10. Decomposer Worker

```go
// internal/core/decomposer.go

func RunDecomposer(humus *Humus, soil *Soil) {
    humus.Decompose(func(compost Compost) {
        switch compost.Action {
        case "create":
            soil.Bury(compost.Entity, compost.Data, 0)
        case "update":
            _, rev, _ := soil.Dig(compost.Entity)
            soil.Bury(compost.Entity, compost.Data, rev)
        case "delete":
            soil.Delete(compost.Entity)
        }
    })
}
```

### 11. Docker Compose

```yaml
version: '3.8'
services:
  nats:
    image: nats:latest
    command: ["--jetstream", "--store_dir=/data", "-p", "4222", "-m", "8222"]
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats-data:/data

volumes:
  nats-data:
```

### 12. Main Entry Point

```go
// cmd/forest/main.go

func main() {
    // Connect to NATS
    nc, _ := nats.Connect("nats://localhost:4222")
    js, _ := nc.JetStream()
    
    // Initialize forest layers
    wind := core.NewWind(nc)
    river, _ := core.NewRiver(js)
    humus, _ := core.NewHumus(js)
    soil, _ := core.NewSoil(js)
    
    // Start decomposer
    go core.RunDecomposer(humus, soil)
    
    // Plant trees (edge parsers)
    paymentTreeBase := core.NewBaseTree("payment", wind)
    paymentTree := trees.NewPaymentTree(paymentTreeBase, river)
    
    // Initialize nims (business logic)
    afterSalesBase := core.NewBaseNim("aftersales", wind, humus, soil)
    afterSalesNim := nims.NewAfterSalesNim(afterSalesBase)
    
    // Start everything
    ctx := context.Background()
    paymentTree.Start(ctx)
    afterSalesNim.Start(ctx)
    
    // Wait for shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh
    
    log.Println("Forest shutting down...")
}
```

## Example Flow

Stripe webhook arrives:

```
river: stripe.webhook {"type": "charge.succeeded", "data": {...}}
            ↓
        PaymentTree.parseStripe()
            ↓
leaf:   payment.completed {customer_id: "X", amount: 99.00, item_id: "jacket"}
            ↓
        wind carries leaf
            ↓
        AfterSalesNim.Handle()
            ↓
humus:  {entity: "tasks/followup-X", action: "create", data: {...}}
            ↓
        Decomposer
            ↓
soil:   tasks/followup-X = {customer_id: "X", due_date: "...", status: "pending"}
            ↓
leaf:   comms.email.send {customer_id: "X", ...}
            ↓
        (CommsNim would catch this)
```

## Build Order

1. `docker-compose.yml` - get NATS running
2. `internal/core/leaf.go` - leaf type
3. `internal/core/wind.go` - pub/sub
4. `internal/core/river.go` - external data stream
5. `internal/core/soil.go` - KV state
6. `internal/core/humus.go` - state change stream
7. `internal/core/tree.go` - base tree (pattern matcher)
8. `internal/core/nim.go` - base nim (business logic)
9. `internal/leaves/types.go` - structured event types
10. `internal/trees/payment.go` - example tree
11. `internal/nims/aftersales.go` - example nim
12. `cmd/forest/main.go` - wire it up
13. Test end-to-end

## Key Principles

- **Trees** parse and structure, no business logic
- **Nims** hold business logic, no parsing
- **Leaves** are the contract between trees and nims
- **Wind** is ephemeral coordination
- **Humus** is persistent commitment
- **Soil** is current truth
- Ordering via JetStream slots
- Concurrency via optimistic locking on soil
