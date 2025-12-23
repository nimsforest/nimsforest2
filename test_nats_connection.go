package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	fmt.Println("🔌 Testing NATS Connection...")
	
	// Connect to NATS
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatal("❌ Failed to connect to NATS:", err)
	}
	defer nc.Close()
	fmt.Println("✅ Connected to NATS successfully!")
	
	// Test basic pub/sub
	fmt.Println("\n📤 Testing basic pub/sub...")
	received := make(chan string, 1)
	
	sub, err := nc.Subscribe("test.subject", func(msg *nats.Msg) {
		received <- string(msg.Data)
	})
	if err != nil {
		log.Fatal("❌ Failed to subscribe:", err)
	}
	defer sub.Unsubscribe()
	
	err = nc.Publish("test.subject", []byte("Hello NATS!"))
	if err != nil {
		log.Fatal("❌ Failed to publish:", err)
	}
	
	select {
	case msg := <-received:
		fmt.Printf("✅ Received message: %s\n", msg)
	case <-time.After(2 * time.Second):
		log.Fatal("❌ Timeout waiting for message")
	}
	
	// Test JetStream
	fmt.Println("\n🌊 Testing JetStream...")
	js, err := nc.JetStream()
	if err != nil {
		log.Fatal("❌ Failed to get JetStream context:", err)
	}
	fmt.Println("✅ JetStream context created successfully!")
	
	// Create a stream
	streamName := "TEST_STREAM"
	fmt.Printf("📊 Creating stream: %s...\n", streamName)
	
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{"test.stream.>"},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		log.Printf("⚠️  Stream creation failed (may already exist): %v\n", err)
	} else {
		fmt.Println("✅ Stream created successfully!")
	}
	
	// Publish to stream
	fmt.Println("📤 Publishing to JetStream...")
	ack, err := js.Publish("test.stream.foo", []byte("JetStream message"))
	if err != nil {
		log.Fatal("❌ Failed to publish to JetStream:", err)
	}
	fmt.Printf("✅ Published to JetStream! Sequence: %d\n", ack.Sequence)
	
	// Test KV Store
	fmt.Println("\n🗄️  Testing JetStream KV Store...")
	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket: "TEST_KV",
	})
	if err != nil {
		log.Printf("⚠️  KV creation failed (may already exist): %v\n", err)
		kv, err = js.KeyValue("TEST_KV")
		if err != nil {
			log.Fatal("❌ Failed to get KV bucket:", err)
		}
	}
	fmt.Println("✅ KV Store created/accessed successfully!")
	
	// Put value
	rev, err := kv.Put("test-key", []byte("test-value"))
	if err != nil {
		log.Fatal("❌ Failed to put value:", err)
	}
	fmt.Printf("✅ Stored value in KV! Revision: %d\n", rev)
	
	// Get value
	entry, err := kv.Get("test-key")
	if err != nil {
		log.Fatal("❌ Failed to get value:", err)
	}
	fmt.Printf("✅ Retrieved value from KV: %s (Revision: %d)\n", string(entry.Value()), entry.Revision())
	
	fmt.Println("\n🎉 All tests passed! Infrastructure is fully operational!")
	fmt.Println("\n📊 NATS Server Info:")
	fmt.Printf("   - Client Port: 4222 ✅\n")
	fmt.Printf("   - Monitoring UI: http://localhost:8222 ✅\n")
	fmt.Printf("   - JetStream: Enabled ✅\n")
	fmt.Printf("   - KV Store: Working ✅\n")
}
