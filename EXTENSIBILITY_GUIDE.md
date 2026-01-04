# 🌲 NimsForest Extensibility Guide

## Overview

NimsForest is designed to be extended with your own custom components. This guide shows you how to add your own **Trees** (data parsers) and **Nims** (business logic processors).

## 🎯 Quick Start

The system now includes **template components** you can copy and customize:

- **`internal/trees/general.go`** - Template tree for parsing external data
- **`internal/nims/general.go`** - Template nim for business logic

## 🌳 Creating Your Own Tree

Trees parse unstructured data from external sources and emit structured leaves.

### Step 1: Copy the Template

```bash
cp internal/trees/general.go internal/trees/your_tree.go
```

### Step 2: Customize the Patterns

Change what river subjects your tree watches:

```go
func (t *YourTree) Patterns() []string {
    // Examples:
    return []string{"river.crm.salesforce.>"}     // CRM webhooks
    // return []string{"river.iot.sensors.>"}     // IoT sensor data
    // return []string{"river.api.external.>"}    // API events
}
```

### Step 3: Implement Parsing Logic

Update the parsing function to handle your data format:

```go
func (t *YourTree) parseYourData(data core.RiverData) {
    // Parse your specific format (JSON, XML, CSV, etc.)
    var payload YourDataType
    json.Unmarshal(data.Data, &payload)

    // Extract relevant fields
    // Make business decisions
    // Emit appropriate leaves

    t.emitYourLeaf(payload)
}
```

### Step 4: Emit Domain-Specific Leaves

```go
func (t *YourTree) emitYourLeaf(data YourDataType) {
    leafData := map[string]interface{}{
        "customer_id": data.CustomerID,
        "event_type":  "customer.updated",
        // ... your fields
    }

    jsonData, _ := json.Marshal(leafData)
    leaf := *core.NewLeaf("customer.updated", jsonData, t.Name())
    t.Drop(leaf)
}
```

### Step 5: Register in main.go

```go
// In cmd/forest/main.go, after other trees:
yourTree := trees.NewYourTree(wind, river)
if err := yourTree.Start(ctx); err != nil {
    log.Fatalf("Failed to start your tree: %v", err)
}
defer yourTree.Stop()
```

## 🧚 Creating Your Own Nim

Nims contain business logic that reacts to leaves (events).

### Step 1: Copy the Template

```bash
cp internal/nims/general.go internal/nims/your_nim.go
```

### Step 2: Define What Leaves to Catch

```go
func (n *YourNim) Subjects() []string {
    // Catch specific events relevant to your domain
    return []string{
        "customer.updated",
        "customer.created",
        "order.placed",
    }
}
```

### Step 3: Implement Business Logic

```go
func (n *YourNim) Handle(ctx context.Context, leaf core.Leaf) error {
    switch leaf.Subject {
    case "customer.updated":
        return n.handleCustomerUpdate(ctx, leaf)
    case "order.placed":
        return n.handleOrderPlaced(ctx, leaf)
    }
    return nil
}

func (n *YourNim) handleCustomerUpdate(ctx context.Context, leaf core.Leaf) error {
    // Parse the leaf data
    var customer Customer
    json.Unmarshal(leaf.Data, &customer)

    // YOUR BUSINESS LOGIC HERE
    // - Read state from Soil via n.Dig()
    // - Make decisions
    // - Persist changes via n.Compost()
    // - Emit new leaves via n.Leaf()

    return nil
}
```

### Step 4: Use State Management

**Read State:**
```go
// Read from Soil
data, revision, err := n.Dig("customer-123")
if err != nil {
    // Handle not found
}
```

**Write State:**
```go
// Write to Humus (decomposer applies to Soil)
customerData, _ := json.Marshal(customer)
slot, err := n.Compost("customer-123", "update", customerData)
```

**Emit Events:**
```go
// Emit leaves for downstream processing
eventData, _ := json.Marshal(YourEvent{...})
err := n.Leaf("your.event", eventData)
```

### Step 5: Register in main.go

```go
// In cmd/forest/main.go, after other nims:
yourNim := nims.NewYourNim(wind, humus, soil)
if err := yourNim.Start(ctx); err != nil {
    log.Fatalf("Failed to start your nim: %v", err)
}
defer yourNim.Stop()
```

## 📋 Real-World Examples

### Example 1: Inventory Management

**Tree**: Watches `river.warehouse.>` for stock updates  
**Nim**: Catches `order.placed` and `inventory.updated`  
**Logic**: Decrements stock, checks reorder thresholds, emits `inventory.low`

### Example 2: Billing System

**Tree**: Watches `river.stripe.>` for payment webhooks  
**Nim**: Catches `subscription.created`, `subscription.renewed`  
**Logic**: Calculates charges, creates invoices, emits `invoice.created`

### Example 3: CRM Integration

**Tree**: Watches `river.crm.salesforce.>` for contact updates  
**Nim**: Catches `contact.created`, `opportunity.updated`  
**Logic**: Syncs data, triggers workflows, emits `task.required`

### Example 4: IoT Monitoring

**Tree**: Watches `river.iot.sensors.>` for sensor readings  
**Nim**: Catches `sensor.reading`  
**Logic**: Checks thresholds, aggregates metrics, emits `sensor.alert`

### Example 5: Analytics

**Tree**: Not needed (catches existing leaves)  
**Nim**: Catches `*.completed`, `*.failed` (wildcards!)  
**Logic**: Counts events, calculates metrics, stores aggregates

## 🧪 Testing Your Components

### 1. Send Test Data

```bash
# Test your tree
nats pub river.your.subject '{"your": "data"}'

# Watch the logs to see:
# - Tree parsing the data
# - Leaves being emitted
# - Nims catching and processing
# - State changes in Soil
```

### 2. Check State

```bash
# View what's in Soil (requires nats CLI)
nats kv get SOIL your-entity-key
```

### 3. Monitor Streams

```bash
# See what flowed through River
nats stream view RIVER

# See state changes in Humus
nats stream view HUMUS
```

## 🎨 Best Practices

### Trees Should:
- ✅ Be stateless (no state management)
- ✅ Focus on parsing and validation
- ✅ Emit one or more leaves per input
- ✅ Handle parsing errors gracefully
- ✅ Log what they're doing

### Nims Should:
- ✅ Contain business logic
- ✅ Be idempotent (can replay safely)
- ✅ Use Compost() for all state changes
- ✅ Emit leaves for downstream processes
- ✅ Handle errors without crashing

### General Tips:
- 🔍 Use wildcards in subjects wisely (`>` matches all remaining, `*` matches one token)
- 📝 Log important decisions and actions
- 🧪 Write unit tests for parsing and business logic
- 🔄 Design for replays (Humus can replay all state changes)
- 🎯 Keep each nim focused on one domain area

## 🔗 Data Flow Example

Let's trace a complete flow:

```
1. External System
   └─> Webhook/API call

2. River (JetStream Stream)
   └─> Persists raw data: river.your.webhook

3. YourTree (Parser)
   └─> Watches: river.your.>
   └─> Parses webhook
   └─> Emits: customer.created leaf

4. Wind (NATS Pub/Sub)
   └─> Distributes leaf to subscribers

5. YourNim (Business Logic)
   └─> Catches: customer.created
   └─> Reads existing data from Soil
   └─> Makes business decisions
   └─> Composts: customer:123 state change → Humus
   └─> Emits: welcome.email.required leaf

6. Decomposer (Background Worker)
   └─> Reads from Humus
   └─> Applies changes to Soil

7. Soil (KV Store)
   └─> Now contains: customer:123 current state

8. EmailNim (Another Nim)
   └─> Catches: welcome.email.required
   └─> Sends welcome email
   └─> Composts: email:456 sent record
```

## 📚 Additional Resources

- **Main README**: Complete system overview
- **internal/trees/payment.go**: Real payment processing example
- **internal/nims/aftersales.go**: Real followup task example
- **internal/trees/general.go**: Template with full documentation
- **internal/nims/general.go**: Template with business logic patterns

## 🚀 Next Steps

1. **Run the demo**: `DEMO=true ./forest`
2. **Copy a template**: Start with general.go
3. **Customize for your domain**: Change patterns and logic
4. **Test with real data**: Send events and watch the flow
5. **Chain multiple nims**: Create complex workflows

## 💡 Remember

> **Trees** stand at the edge, converting messy external data into clean events.  
> **Nims** live in the core, making decisions and managing state.  
> Together, they create a flexible, event-driven system that scales!

---

Need help? Check the inline documentation in the template files or look at the existing payment/aftersales examples.
