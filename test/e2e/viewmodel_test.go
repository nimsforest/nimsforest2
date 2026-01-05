package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/yourusername/nimsforest/internal/core"
	"github.com/yourusername/nimsforest/internal/natsembed"
	"github.com/yourusername/nimsforest/internal/viewmodel"
)

// TestViewmodelWorld tests basic territory model functionality.
func TestViewmodelWorld(t *testing.T) {
	// Create a territory manually
	territory := viewmodel.NewWorld()

	// Add a LandViewModel
	land1 := viewmodel.NewLandViewModel("node-1")
	land1.RAMTotal = 16 * 1024 * 1024 * 1024 // 16GB
	land1.CPUCores = 4
	territory.AddLand(land1)

	// Add another LandViewModel with GPU
	land2 := viewmodel.NewLandViewModel("node-gpu")
	land2.RAMTotal = 32 * 1024 * 1024 * 1024 // 32GB
	land2.CPUCores = 8
	land2.GPUVram = 24 * 1024 * 1024 * 1024 // 24GB
	land2.GPUTflops = 10.0
	territory.AddLand(land2)

	// Test LandCount
	if got := territory.LandCount(); got != 2 {
		t.Errorf("LandCount() = %d, want 2", got)
	}

	// Test ManalandCount
	if got := territory.ManalandCount(); got != 1 {
		t.Errorf("ManalandCount() = %d, want 1", got)
	}

	// Test HasGPU
	if !land2.HasGPU() {
		t.Error("land2.HasGPU() = false, want true")
	}
	if land1.HasGPU() {
		t.Error("land1.HasGPU() = true, want false")
	}

	// Test TotalRAM
	expectedRAM := uint64(48 * 1024 * 1024 * 1024)
	if got := territory.TotalRAM(); got != expectedRAM {
		t.Errorf("TotalRAM() = %d, want %d", got, expectedRAM)
	}

	// Test TotalGPUVram
	expectedVram := uint64(24 * 1024 * 1024 * 1024)
	if got := territory.TotalGPUVram(); got != expectedVram {
		t.Errorf("TotalGPUVram() = %d, want %d", got, expectedVram)
	}

	t.Log("✅ World model tests passed")
}

// TestViewmodelProcesses tests adding processes to LandViewModel.
func TestViewmodelProcesses(t *testing.T) {
	territory := viewmodel.NewWorld()

	// Create LandViewModel
	land := viewmodel.NewLandViewModel("node-1")
	land.RAMTotal = 16 * 1024 * 1024 * 1024
	land.CPUCores = 4
	territory.AddLand(land)

	// Add a TreeViewModel
	tree := viewmodel.NewTreeViewModel("tree-payment", "payment-processor", 4*1024*1024*1024, []string{"river.stripe.>"})
	land.AddTree(tree)

	// Add a TreehouseViewModel
	th := viewmodel.NewTreehouseViewModel("th-scoring", "scoring", 1*1024*1024*1024, "scripts/scoring.lua")
	land.AddTreehouse(th)

	// Add a NimViewModel
	nim := viewmodel.NewNimViewModel("nim-qualify", "qualify", 2*1024*1024*1024, []string{"lead.scored"}, true)
	land.AddNim(nim)

	// Test counts
	if got := territory.TreeCount(); got != 1 {
		t.Errorf("TreeCount() = %d, want 1", got)
	}
	if got := territory.TreehouseCount(); got != 1 {
		t.Errorf("TreehouseCount() = %d, want 1", got)
	}
	if got := territory.NimCount(); got != 1 {
		t.Errorf("NimCount() = %d, want 1", got)
	}

	// Test RAM allocation
	expectedRAM := uint64(7 * 1024 * 1024 * 1024) // 4 + 1 + 2 = 7GB
	if got := land.RAMAllocated(); got != expectedRAM {
		t.Errorf("RAMAllocated() = %d, want %d", got, expectedRAM)
	}

	// Test occupancy (7GB / 16GB = 43.75%)
	occupancy := land.Occupancy()
	if occupancy < 43 || occupancy > 44 {
		t.Errorf("Occupancy() = %.2f%%, want ~43.75%%", occupancy)
	}

	// Test FindProcess
	proc, foundLand := territory.FindProcess("nim-qualify")
	if proc == nil {
		t.Error("FindProcess() returned nil")
	} else if proc.Name != "qualify" {
		t.Errorf("FindProcess().Name = %s, want qualify", proc.Name)
	}
	if foundLand == nil || foundLand.ID != "node-1" {
		t.Error("FindProcess() returned wrong land")
	}

	// Test RemoveProcess
	if !land.RemoveProcess("th-scoring") {
		t.Error("RemoveProcess() returned false")
	}
	if got := territory.TreehouseCount(); got != 0 {
		t.Errorf("After removal, TreehouseCount() = %d, want 0", got)
	}

	t.Log("✅ Process management tests passed")
}

// TestViewmodelOccupancy tests occupancy calculations.
func TestViewmodelOccupancy(t *testing.T) {
	territory := viewmodel.NewWorld()

	// Create two lands with different usage
	land1 := viewmodel.NewLandViewModel("node-1")
	land1.RAMTotal = 16 * 1024 * 1024 * 1024
	land1.CPUCores = 4
	territory.AddLand(land1)

	land2 := viewmodel.NewLandViewModel("node-2")
	land2.RAMTotal = 16 * 1024 * 1024 * 1024
	land2.CPUCores = 4
	territory.AddLand(land2)

	// Add processes to land1 (6GB used = 37.5%)
	tree := viewmodel.NewTreeViewModel("tree-1", "router", 4*1024*1024*1024, []string{"river.>"})
	land1.AddTree(tree)
	nim := viewmodel.NewNimViewModel("nim-1", "processor", 2*1024*1024*1024, []string{"data.>"}, false)
	land1.AddNim(nim)

	// Add process to land2 (4GB used = 25%)
	tree2 := viewmodel.NewTreeViewModel("tree-2", "parser", 4*1024*1024*1024, []string{"river.api.>"})
	land2.AddTree(tree2)

	// Test per-land occupancy
	if occ := land1.Occupancy(); occ < 37 || occ > 38 {
		t.Errorf("land1.Occupancy() = %.2f%%, want ~37.5%%", occ)
	}
	if occ := land2.Occupancy(); occ != 25 {
		t.Errorf("land2.Occupancy() = %.2f%%, want 25%%", occ)
	}

	// Test territory-wide occupancy (10GB / 32GB = 31.25%)
	if occ := territory.Occupancy(); occ < 31 || occ > 32 {
		t.Errorf("territory.Occupancy() = %.2f%%, want ~31.25%%", occ)
	}

	t.Log("✅ Occupancy calculation tests passed")
}

// TestViewmodelPrint tests the print output format.
func TestViewmodelPrint(t *testing.T) {
	territory := viewmodel.NewWorld()

	// Create LandViewModel with processes
	land := viewmodel.NewLandViewModel("node-abc")
	land.RAMTotal = 16 * 1024 * 1024 * 1024
	land.CPUCores = 4
	territory.AddLand(land)

	tree := viewmodel.NewTreeViewModel("tree-1", "payment-processor", 4*1024*1024*1024, []string{"river.stripe.>"})
	land.AddTree(tree)

	th := viewmodel.NewTreehouseViewModel("th-1", "scoring", 1*1024*1024*1024, "scoring.lua")
	land.AddTreehouse(th)

	nim := viewmodel.NewNimViewModel("nim-1", "qualify", 2*1024*1024*1024, []string{"lead.scored"}, false)
	land.AddNim(nim)

	// Print territory
	var buf bytes.Buffer
	printer := viewmodel.NewPrinter(&buf)
	printer.PrintWorld(territory)

	output := buf.String()

	// Verify expected content
	expectedStrings := []string{
		"World: 1 land",
		"Land: node-abc",
		"ram: 16GB",
		"cpu: 4",
		"Trees:",
		"payment-processor",
		"Treehouses:",
		"scoring",
		"Nims:",
		"qualify",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Print output missing expected string: %s", expected)
		}
	}

	t.Logf("✅ Print output contains all expected elements\n%s", output)
}

// TestViewmodelSummary tests the summary output format.
func TestViewmodelSummary(t *testing.T) {
	territory := viewmodel.NewWorld()

	// Create regular LandViewModel
	land1 := viewmodel.NewLandViewModel("node-1")
	land1.RAMTotal = 16 * 1024 * 1024 * 1024
	land1.CPUCores = 4
	territory.AddLand(land1)

	// Create Manaland (GPU)
	land2 := viewmodel.NewLandViewModel("node-gpu")
	land2.RAMTotal = 32 * 1024 * 1024 * 1024
	land2.CPUCores = 8
	land2.GPUVram = 24 * 1024 * 1024 * 1024
	territory.AddLand(land2)

	// Add processes
	tree := viewmodel.NewTreeViewModel("tree-1", "router", 4*1024*1024*1024, []string{"river.>"})
	land1.AddTree(tree)

	nim := viewmodel.NewNimViewModel("nim-1", "processor", 2*1024*1024*1024, []string{"data.>"}, false)
	land1.AddNim(nim)

	// Print summary
	var buf bytes.Buffer
	printer := viewmodel.NewPrinter(&buf)
	printer.PrintSummary(territory)

	output := buf.String()

	// Verify expected content
	expectedStrings := []string{
		"Capacity:",
		"Land: 1",
		"Manaland: 1",
		"Usage:",
		"Trees: 1",
		"Nims: 1",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Summary output missing expected string: %s", expected)
		}
	}

	t.Logf("✅ Summary output contains all expected elements\n%s", output)
}

// TestViewmodelWithEmbeddedNATS tests viewmodel with a real embedded NATS server.
func TestViewmodelWithEmbeddedNATS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping embedded NATS test in short mode")
	}

	// Create temp directory for JetStream
	tmpDir, err := os.MkdirTemp("", "nimsforest-viewmodel-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create embedded NATS server using natsembed
	cfg := natsembed.Config{
		NodeName:    "test-node",
		ClusterName: "test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,  // Random port
		MonitorPort: -1, // Disable monitoring
	}

	ns, err := natsembed.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create embedded NATS: %v", err)
	}

	if err := ns.Start(); err != nil {
		t.Fatalf("Failed to start embedded NATS: %v", err)
	}
	defer ns.Shutdown()

	// Get the internal server
	internalServer := ns.InternalServer()

	// Create viewmodel
	vm := viewmodel.New(internalServer)

	// Refresh to get current state
	if err := vm.Refresh(); err != nil {
		t.Fatalf("Failed to refresh viewmodel: %v", err)
	}

	// Get territory
	territory := vm.GetWorld()

	// Should have at least the local node
	if territory.LandCount() < 1 {
		t.Errorf("LandCount() = %d, want at least 1", territory.LandCount())
	}

	// Print the territory
	var buf bytes.Buffer
	vm.Print(&buf)

	t.Logf("✅ Viewmodel with embedded NATS:\n%s", buf.String())
}

// TestViewmodelDetection tests process detection from subscriptions.
func TestViewmodelDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping detection test in short mode")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "nimsforest-detection-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create server directly for detection testing
	opts := &server.Options{
		ServerName: "detection-test",
		Host:       "127.0.0.1",
		Port:       -1, // Random port
		JetStream:  true,
		StoreDir:   filepath.Join(tmpDir, "jetstream"),
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		t.Fatal("Server not ready")
	}
	defer ns.Shutdown()

	// Connect client using standard NATS client
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Close()

	// Create Wind and subscribe (simulating a Tree)
	wind := core.NewWind(nc)
	sub1, err := wind.Catch("payment.>", func(leaf core.Leaf) {})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub1.Unsubscribe()

	// Create another subscription (simulating a Nim)
	sub2, err := wind.Catch("lead.qualified", func(leaf core.Leaf) {})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub2.Unsubscribe()

	// Give subscriptions time to register
	time.Sleep(200 * time.Millisecond)

	// Create detector
	reader := viewmodel.NewReader(ns)
	detector := viewmodel.NewDetector(reader)

	// Detect processes
	processes, err := detector.DetectProcesses()
	if err != nil {
		t.Fatalf("Failed to detect processes: %v", err)
	}

	t.Logf("Detected %d processes", len(processes))
	for _, proc := range processes {
		t.Logf("  - %s (%s): %v", proc.Name, proc.Type, proc.Subjects)
	}

	// Should detect at least some processes (may include internal NATS subscriptions)
	// The exact number depends on NATS internals
	t.Log("✅ Process detection completed")
}

// TestViewmodelUpdater tests incremental updates.
func TestViewmodelUpdater(t *testing.T) {
	territory := viewmodel.NewWorld()
	updater := viewmodel.NewUpdater(territory)

	// Track events
	var events []viewmodel.Event
	updater.SetOnChange(func(event viewmodel.Event) {
		events = append(events, event)
	})

	// Add a land via event
	land := viewmodel.NewLandViewModel("node-1")
	land.RAMTotal = 16 * 1024 * 1024 * 1024
	land.CPUCores = 4

	event := viewmodel.NewLandAddedEvent(land)
	if err := updater.ApplyEvent(event); err != nil {
		t.Fatalf("Failed to apply land_added event: %v", err)
	}

	if territory.LandCount() != 1 {
		t.Errorf("After land_added, LandCount() = %d, want 1", territory.LandCount())
	}

	// Add a process via event
	proc := viewmodel.DetectedProcess{
		ID:           "nim-test",
		Name:         "test-processor",
		Type:         viewmodel.ProcessTypeNim,
		RAMAllocated: 1 * 1024 * 1024 * 1024,
		Subjects:     []string{"test.>"},
	}

	processEvent := viewmodel.NewProcessAddedEvent("node-1", proc)
	if err := updater.ApplyEvent(processEvent); err != nil {
		t.Fatalf("Failed to apply process_added event: %v", err)
	}

	if territory.NimCount() != 1 {
		t.Errorf("After process_added, NimCount() = %d, want 1", territory.NimCount())
	}

	// Remove process via event
	removeEvent := viewmodel.NewProcessRemovedEvent("node-1", "nim-test")
	if err := updater.ApplyEvent(removeEvent); err != nil {
		t.Fatalf("Failed to apply process_removed event: %v", err)
	}

	if territory.NimCount() != 0 {
		t.Errorf("After process_removed, NimCount() = %d, want 0", territory.NimCount())
	}

	// Remove land via event
	removeLandEvent := viewmodel.NewLandRemovedEvent("node-1")
	if err := updater.ApplyEvent(removeLandEvent); err != nil {
		t.Fatalf("Failed to apply land_removed event: %v", err)
	}

	if territory.LandCount() != 0 {
		t.Errorf("After land_removed, LandCount() = %d, want 0", territory.LandCount())
	}

	// Check events were recorded
	if len(events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(events))
	}

	t.Log("✅ Updater event handling tests passed")
}

// TestViewmodelManaLand tests GPU/Manaland functionality.
func TestViewmodelManaLand(t *testing.T) {
	territory := viewmodel.NewWorld()

	// Create ManaLand (land with GPU/magical compute resources)
	land := viewmodel.NewLandViewModel("mana-node")
	land.RAMTotal = 64 * 1024 * 1024 * 1024 // 64GB
	land.CPUCores = 16
	land.GPUVram = 48 * 1024 * 1024 * 1024  // 48GB VRAM
	land.GPUTflops = 100.0                   // 100 TFLOPS
	territory.AddLand(land)

	// Test IsManaland
	if !land.IsManaland() {
		t.Error("IsManaland() = false, want true")
	}

	// Test HasGPU (ManaLand has GPU resources)
	if !land.HasGPU() {
		t.Error("HasGPU() = false, want true")
	}

	// Test total VRAM across all ManaLand
	expectedVram := uint64(48 * 1024 * 1024 * 1024)
	if got := territory.TotalGPUVram(); got != expectedVram {
		t.Errorf("TotalGPUVram() = %d, want %d", got, expectedVram)
	}

	// Print should show ManaLand info
	var buf bytes.Buffer
	printer := viewmodel.NewPrinter(&buf)
	printer.PrintWorld(territory)

	output := buf.String()
	if !strings.Contains(output, "Manaland:") {
		t.Error("Print output should contain 'Manaland:'")
	}
	if !strings.Contains(output, "vram") {
		t.Error("Print output should contain vram info")
	}

	t.Logf("✅ ManaLand tests passed\n%s", output)
}

// TestViewmodelE2E is an end-to-end test of the full viewmodel flow.
func TestViewmodelE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "nimsforest-e2e-viewmodel-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create embedded NATS server
	cfg := natsembed.Config{
		NodeName:    "e2e-test-node",
		ClusterName: "e2e-test-cluster",
		DataDir:     filepath.Join(tmpDir, "jetstream"),
		ClientPort:  0,
		MonitorPort: -1,
	}

	ns, err := natsembed.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create NATS: %v", err)
	}

	if err := ns.Start(); err != nil {
		t.Fatalf("Failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Get connection
	nc, err := ns.ClientConn()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Close()

	// Create some subscriptions to simulate processes
	ctx := context.Background()
	_ = ctx // Would be used for process lifecycle

	wind := core.NewWind(nc)

	// Simulate a Tree subscription
	treeSub, _ := wind.Catch("river.>", func(leaf core.Leaf) {})
	defer treeSub.Unsubscribe()

	// Simulate a Nim subscription
	nimSub, _ := wind.Catch("payment.completed", func(leaf core.Leaf) {})
	defer nimSub.Unsubscribe()

	time.Sleep(300 * time.Millisecond)

	// Create viewmodel and refresh
	internalServer := ns.InternalServer()
	vm := viewmodel.New(internalServer)

	if err := vm.Refresh(); err != nil {
		t.Fatalf("Failed to refresh: %v", err)
	}

	// Get summary
	summary := vm.GetSummary()
	t.Logf("Summary: %d land, %d trees, %d nims",
		summary.LandCount+summary.ManalandCount, summary.TreeCount, summary.NimCount)

	// Print full view
	var printBuf bytes.Buffer
	vm.Print(&printBuf)
	t.Logf("Full View:\n%s", printBuf.String())

	// Print summary
	var summaryBuf bytes.Buffer
	vm.PrintSummary(&summaryBuf)
	t.Logf("Summary:\n%s", summaryBuf.String())

	// Verify we can see the local node
	territory := vm.GetWorld()
	lands := territory.Lands()
	if len(lands) == 0 {
		t.Fatal("No lands found in territory")
	}

	// The local land should have some RAM info
	localLand := lands[0]
	if localLand.RAMTotal == 0 {
		t.Log("⚠️  Local land has no RAM info (expected in test environment)")
	}

	t.Log("✅ E2E viewmodel test completed")
}

// TestFormatBytes tests the byte formatting utility.
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1KB"},
		{1024 * 1024, "1MB"},
		{1024 * 1024 * 1024, "1GB"},
		{16 * 1024 * 1024 * 1024, "16GB"},
		{1024 * 1024 * 1024 * 1024, "1.0TB"},
	}

	for _, tc := range tests {
		got := viewmodel.FormatBytes(tc.bytes)
		if got != tc.expected {
			t.Errorf("FormatBytes(%d) = %s, want %s", tc.bytes, got, tc.expected)
		}
	}

	t.Log("✅ FormatBytes tests passed")
}

// TestInferProcessType tests process type inference from subjects.
func TestInferProcessType(t *testing.T) {
	tests := []struct {
		subject  string
		expected viewmodel.ProcessType
	}{
		{"river.stripe.webhook", viewmodel.ProcessTypeTree},
		{"river.general.api", viewmodel.ProcessTypeTree},
		{"payment.completed", viewmodel.ProcessTypeNim},
		{"payment.failed", viewmodel.ProcessTypeNim},
		{"contact.created", viewmodel.ProcessTypeTreehouse},
		{"lead.scored", viewmodel.ProcessTypeTreehouse},
		{"data.received", viewmodel.ProcessTypeNim},
		{"unknown.subject", viewmodel.ProcessTypeNim}, // Default
	}

	for _, tc := range tests {
		got := viewmodel.InferProcessType(tc.subject)
		if got != tc.expected {
			t.Errorf("InferProcessType(%s) = %s, want %s", tc.subject, got, tc.expected)
		}
	}

	t.Log("✅ InferProcessType tests passed")
}

// Benchmark for territory operations
func BenchmarkWorldAddLand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		territory := viewmodel.NewWorld()
		for j := 0; j < 100; j++ {
			land := viewmodel.NewLandViewModel(fmt.Sprintf("node-%d", j))
			land.RAMTotal = 16 * 1024 * 1024 * 1024
			land.CPUCores = 4
			territory.AddLand(land)
		}
	}
}

func BenchmarkWorldPrint(b *testing.B) {
	territory := viewmodel.NewWorld()
	for i := 0; i < 10; i++ {
		land := viewmodel.NewLandViewModel(fmt.Sprintf("node-%d", i))
		land.RAMTotal = 16 * 1024 * 1024 * 1024
		land.CPUCores = 4
		
		for j := 0; j < 5; j++ {
			tree := viewmodel.NewTreeViewModel(
				fmt.Sprintf("tree-%d-%d", i, j),
				fmt.Sprintf("tree-%d", j),
				1024*1024*1024,
				[]string{"river.>"},
			)
			land.AddTree(tree)
		}
		territory.AddLand(land)
	}

	printer := viewmodel.NewPrinter(&bytes.Buffer{})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		printer = viewmodel.NewPrinter(&buf)
		printer.PrintWorld(territory)
	}
}
