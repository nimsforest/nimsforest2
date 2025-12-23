package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to get JetStream: %v", err)
	}

	fmt.Println("✅ Connected to NATS")
	fmt.Println("📨 Sending test Stripe webhooks...")
	fmt.Println()

	// Test 1: Successful high-value payment
	webhook1 := `{
		"type": "charge.succeeded",
		"data": {
			"object": {
				"id": "ch_demo_123",
				"amount": 15000,
				"currency": "usd",
				"customer": "cus_demo_alice",
				"metadata": {
					"item_id": "premium-jacket"
				}
			}
		}
	}`

	_, err = js.Publish("river.stripe.webhook", []byte(webhook1))
	if err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}
	fmt.Println("✅ Sent successful payment webhook ($150.00)")
	time.Sleep(500 * time.Millisecond)

	// Test 2: Failed payment
	webhook2 := `{
		"type": "charge.failed",
		"data": {
			"object": {
				"id": "ch_demo_456",
				"amount": 5000,
				"currency": "usd",
				"customer": "cus_demo_bob",
				"failure_message": "insufficient_funds",
				"metadata": {
					"item_id": "basic-tee"
				}
			}
		}
	}`

	_, err = js.Publish("river.stripe.webhook", []byte(webhook2))
	if err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}
	fmt.Println("✅ Sent failed payment webhook ($50.00)")
	time.Sleep(500 * time.Millisecond)

	// Test 3: Low-value payment (no email)
	webhook3 := `{
		"type": "charge.succeeded",
		"data": {
			"object": {
				"id": "ch_demo_789",
				"amount": 2500,
				"currency": "usd",
				"customer": "cus_demo_charlie",
				"metadata": {
					"item_id": "basic-tee"
				}
			}
		}
	}`

	_, err = js.Publish("river.stripe.webhook", []byte(webhook3))
	if err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}
	fmt.Println("✅ Sent low-value payment webhook ($25.00)")
	
	fmt.Println()
	fmt.Println("📊 Test data sent! Check the forest application logs to see the processing flow.")
	fmt.Println()
}
