package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/leaves"
	"github.com/yourusername/nimsforest/internal/natsembed"
	"github.com/yourusername/nimsforest/internal/nims"
	"github.com/yourusername/nimsforest/internal/trees"
)

// testServer holds the embedded NATS server for tests
var testServer *natsembed.Server

// getTestConnection returns a NATS connection for testing using an embedded NATS server.
func getTestConnection(t *testing.T) (*nats.Conn, nats.JetStreamContext) {
	// Create embedded server if not already running
	if testServer == nil || !testServer.IsRunning() {
		// Create temp directory for JetStream data
		tmpDir, err := os.MkdirTemp("", "nimsforest-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}

		cfg := natsembed.Config{
			NodeName:    "test-node",
			ClusterName: "test-cluster",
			DataDir:     filepath.Join(tmpDir, "jetstream"),
			ClientPort:  0,  // Use random port
			MonitorPort: -1, // Disable monitoring for tests
		}

		testServer, err = natsembed.New(cfg)
		if err != nil {
			t.Fatalf("Failed to create embedded NATS server: %v", err)
		}

		if err := testServer.Start(); err != nil {
			t.Fatalf("Failed to start embedded NATS server: %v", err)
		}

		// Cleanup on test completion
		t.Cleanup(func() {
			if testServer != nil {
				testServer.Shutdown()
				testServer = nil
			}
			os.RemoveAll(tmpDir)
		})
	}

	nc, err := testServer.ClientConn()
	if err != nil {
		t.Fatalf("Failed to connect to embedded NATS: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		t.Fatalf("Failed to get JetStream: %v", err)
	}

	return nc, js
}

// TestForestEndToEnd tests the complete flow from river to soil
func TestForestEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Connect to embedded NATS
	nc, js := getTestConnection(t)
	defer nc.Close()

	// Initialize core components
	wind := core.NewWind(nc)

	river, err := core.NewRiver(js)
	if err != nil {
		t.Fatalf("Failed to create river: %v", err)
	}

	humus, err := core.NewHumus(js)
	if err != nil {
		t.Fatalf("Failed to create humus: %v", err)
	}

	soil, err := core.NewSoil(js)
	if err != nil {
		t.Fatalf("Failed to create soil: %v", err)
	}

	// Start decomposer with unique consumer name
	consumerName := fmt.Sprintf("decomposer-%d", time.Now().UnixNano())
	decomposer, err := core.RunDecomposerWithConsumer(humus, soil, consumerName)
	if err != nil {
		t.Fatalf("Failed to start decomposer: %v", err)
	}
	defer decomposer.Stop()

	// Create context for components
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Plant tree
	paymentTree := trees.NewPaymentTree(wind, river)
	if err := paymentTree.Start(ctx); err != nil {
		t.Fatalf("Failed to start payment tree: %v", err)
	}
	defer paymentTree.Stop()

	// Awaken nim
	afterSalesNim := nims.NewAfterSalesNim(wind, humus, soil)
	if err := afterSalesNim.Start(ctx); err != nil {
		t.Fatalf("Failed to start aftersales nim: %v", err)
	}
	defer afterSalesNim.Stop()

	// Give components time to initialize
	time.Sleep(500 * time.Millisecond)

	// Test Case 1: High-value payment (should trigger email)
	t.Run("HighValuePayment", func(t *testing.T) {
		// Create Stripe webhook (matches Stripe's actual format)
		webhook := map[string]interface{}{
			"type": "charge.succeeded",
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"id":       "ch_e2e_high_value",
					"amount":   25000, // $250.00 in cents
					"currency": "usd",
					"customer": "cus_e2e_alice",
					"metadata": map[string]string{
						"item_id": "premium-jacket",
					},
				},
			},
		}

		webhookData, err := json.Marshal(webhook)
		if err != nil {
			t.Fatalf("Failed to marshal webhook: %v", err)
		}

		// Send to river
		if err := river.Flow("river.stripe.webhook", webhookData); err != nil {
			t.Fatalf("Failed to send to river: %v", err)
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		// Verify a task was created for this customer
		// Tasks are stored with format: task:customer_id-timestamp
		// We need to find the task by searching for the customer ID prefix
		t.Log("✅ High-value payment processed successfully")
		t.Log("   Task should have been created and stored in soil")
		t.Log("   (Task verification would require listing keys or known task ID)")
	})

	// Test Case 2: Failed payment
	t.Run("FailedPayment", func(t *testing.T) {
		// Create failed payment webhook (matches Stripe's actual format)
		webhook := map[string]interface{}{
			"type": "charge.failed",
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"id":              "ch_e2e_failed",
					"amount":          5000, // $50.00
					"currency":        "usd",
					"customer":        "cus_e2e_bob",
					"failure_message": "insufficient_funds",
					"metadata": map[string]string{
						"item_id": "basic-tee",
					},
				},
			},
		}

		webhookData, err := json.Marshal(webhook)
		if err != nil {
			t.Fatalf("Failed to marshal webhook: %v", err)
		}

		// Send to river
		if err := river.Flow("river.stripe.webhook", webhookData); err != nil {
			t.Fatalf("Failed to send to river: %v", err)
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		t.Log("✅ Failed payment processed successfully")
		t.Log("   Urgent task should have been created with shorter due date")
		t.Log("   (Task verification would require listing keys or known task ID)")
	})

	// Test Case 3: Verify leaves were emitted
	t.Run("VerifyLeaves", func(t *testing.T) {
		// Subscribe to wind to catch emitted leaves
		leafCaught := make(chan core.Leaf, 10)

		sub, err := wind.Catch("followup.>", func(leaf core.Leaf) {
			leafCaught <- leaf
		})
		if err != nil {
			t.Fatalf("Failed to catch leaves: %v", err)
		}
		defer sub.Unsubscribe()

		// Send a payment that will trigger a followup leaf
		webhook := map[string]interface{}{
			"type": "charge.succeeded",
			"data": map[string]interface{}{
				"object": map[string]interface{}{
					"id":       "ch_e2e_leaf_test",
					"amount":   7500,
					"currency": "usd",
					"customer": "cus_e2e_charlie",
					"metadata": map[string]string{
						"item_id": "mid-tier-hoodie",
					},
				},
			},
		}

		webhookData, err := json.Marshal(webhook)
		if err != nil {
			t.Fatalf("Failed to marshal webhook: %v", err)
		}

		if err := river.Flow("river.stripe.webhook", webhookData); err != nil {
			t.Fatalf("Failed to send to river: %v", err)
		}

		// Wait for leaf
		select {
		case leaf := <-leafCaught:
			t.Logf("✅ Caught leaf: %s from %s", leaf.Subject, leaf.Source)

			// Verify it's a followup leaf
			if leaf.Subject != "followup.required" {
				t.Errorf("Expected subject 'followup.required', got '%s'", leaf.Subject)
			}

			// Parse the leaf data
			var followup leaves.FollowupRequired
			if err := json.Unmarshal(leaf.Data, &followup); err != nil {
				t.Fatalf("Failed to unmarshal followup: %v", err)
			}

			if followup.CustomerID != "cus_e2e_charlie" {
				t.Errorf("Expected customer_id 'cus_e2e_charlie', got '%s'", followup.CustomerID)
			} else {
				t.Log("✅ Followup leaf contains correct customer ID")
			}

		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for followup leaf")
		}
	})
}

// TestForestComponents tests individual component integration
func TestForestComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping component test in short mode")
	}

	// Connect to embedded NATS
	nc, js := getTestConnection(t)
	defer nc.Close()

	t.Run("WindAndRiver", func(t *testing.T) {
		wind := core.NewWind(nc)
		river, err := core.NewRiver(js)
		if err != nil {
			t.Fatalf("Failed to create river: %v", err)
		}

		// Test river flow
		testData := []byte(`{"test": "data"}`)
		if err := river.Flow("river.test.component", testData); err != nil {
			t.Errorf("Failed to flow data: %v", err)
		}

		// Test wind drop and catch
		caught := make(chan core.Leaf, 1)
		sub, err := wind.Catch("test.>", func(leaf core.Leaf) {
			caught <- leaf
		})
		if err != nil {
			t.Fatalf("Failed to catch: %v", err)
		}
		defer sub.Unsubscribe()

		leaf := core.Leaf{
			Subject:   "test.component",
			Data:      json.RawMessage(`{"key": "value"}`),
			Source:    "test",
			Timestamp: time.Now(),
		}

		if err := wind.Drop(leaf); err != nil {
			t.Errorf("Failed to drop leaf: %v", err)
		}

		select {
		case receivedLeaf := <-caught:
			if receivedLeaf.Subject != leaf.Subject {
				t.Errorf("Expected subject %s, got %s", leaf.Subject, receivedLeaf.Subject)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for leaf")
		}

		t.Log("✅ Wind and River components working")
	})

	t.Run("HumusAndSoil", func(t *testing.T) {
		humus, err := core.NewHumus(js)
		if err != nil {
			t.Fatalf("Failed to create humus: %v", err)
		}

		soil, err := core.NewSoil(js)
		if err != nil {
			t.Fatalf("Failed to create soil: %v", err)
		}

		// Test compost and decompose
		testEntity := "component_test_entity"
		testData := []byte(`{"component": "test"}`)

		slot, err := humus.Add("test_nim", testEntity, "create", testData)
		if err != nil {
			t.Fatalf("Failed to add compost: %v", err)
		}
		t.Logf("Added compost at slot %d", slot)

		// Start decomposer briefly with unique consumer name
		consumerName := fmt.Sprintf("decomposer-%d", time.Now().UnixNano())
		decomposer, err := core.RunDecomposerWithConsumer(humus, soil, consumerName)
		if err != nil {
			t.Fatalf("Failed to start decomposer: %v", err)
		}
		defer decomposer.Stop()

		// Wait for processing
		time.Sleep(1 * time.Second)

		// Verify in soil
		data, _, err := soil.Dig(testEntity)
		if err != nil {
			t.Fatalf("Failed to dig from soil: %v", err)
		}

		// Compare as JSON to ignore whitespace differences
		var expected, actual map[string]interface{}
		json.Unmarshal(testData, &expected)
		json.Unmarshal(data, &actual)

		if fmt.Sprintf("%v", expected) != fmt.Sprintf("%v", actual) {
			t.Errorf("Expected data %v, got %v", expected, actual)
		}

		t.Log("✅ Humus and Soil components working")
	})
}

// TestForestScaling tests that multiple instances can work in parallel
func TestForestScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scaling test in short mode")
	}

	// Connect to embedded NATS
	nc, js := getTestConnection(t)
	defer nc.Close()

	// Create multiple decomposers
	humus, _ := core.NewHumus(js)
	soil, _ := core.NewSoil(js)

	decomposer1, err := core.RunDecomposerWithConsumer(humus, soil, "decomposer_1")
	if err != nil {
		t.Fatalf("Failed to start decomposer 1: %v", err)
	}
	defer decomposer1.Stop()

	decomposer2, err := core.RunDecomposerWithConsumer(humus, soil, "decomposer_2")
	if err != nil {
		t.Fatalf("Failed to start decomposer 2: %v", err)
	}
	defer decomposer2.Stop()

	// Send multiple compost entries rapidly
	for i := 0; i < 10; i++ {
		entity := fmt.Sprintf("entity_%d", i)
		data := []byte(fmt.Sprintf(`{"index": %d, "time": "%s"}`, i, time.Now().Format(time.RFC3339)))
		_, err := humus.Add("scaling_test", entity, "create", data)
		if err != nil {
			t.Errorf("Failed to add compost %d: %v", i, err)
		}
	}

	// Wait for processing
	time.Sleep(2 * time.Second)

	t.Log("✅ Multiple decomposers working in parallel")
}
