// Package viewmodel provides a view model for the NimsForest cluster state.
//
// The viewmodel package enables querying and displaying the current state of:
//   - Land (nodes in the NATS cluster)
//   - Trees (data parsers watching the river)
//   - Treehouses (Lua script processors)
//   - Nims (business logic handlers)
//
// # Architecture
//
// The viewmodel consists of several components:
//   - Model: Core data structures (Land, Tree, Treehouse, Nim, World)
//   - Reader: Reads cluster state from the embedded NATS server
//   - Mapper: Converts cluster snapshots to World
//   - Detector: Monitors subscriptions to detect processes
//   - Updater: Applies incremental updates to the World
//   - Printer: Formats World data for CLI output
//
// # Usage
//
// Basic usage to get a snapshot of the current cluster state:
//
//	vm := viewmodel.New(natsServer)
//	territory, err := vm.GetWorld()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	vm.Print(os.Stdout)
//
// For live updates:
//
//	vm := viewmodel.New(natsServer)
//	vm.StartDetection()
//	defer vm.StopDetection()
//	// World is updated automatically as processes come and go
package viewmodel

import (
	"fmt"
	"io"
	"sync"

	"github.com/nats-io/nats-server/v2/server"
)

// ViewModel provides a unified interface to the cluster state.
type ViewModel struct {
	server    *server.Server
	reader    *Reader
	mapper    *Mapper
	detector  *Detector
	updater   *Updater
	territory *World
	mu        sync.RWMutex
}

// New creates a new ViewModel for the given NATS server.
func New(ns *server.Server) *ViewModel {
	reader := NewReader(ns)
	mapper := NewMapper()
	territory := NewWorld()
	detector := NewDetector(reader)
	updater := NewUpdater(territory)

	vm := &ViewModel{
		server:    ns,
		reader:    reader,
		mapper:    mapper,
		detector:  detector,
		updater:   updater,
		territory: territory,
	}

	// Wire up detector callbacks
	detector.SetWorld(territory)
	detector.SetOnProcessAdded(func(proc DetectedProcess) {
		// Find the best land to add this process to
		lands := territory.Lands()
		if len(lands) > 0 {
			event := NewProcessAddedEvent(lands[0].ID, proc)
			updater.ApplyEvent(event)
		}
	})
	detector.SetOnProcessRemoved(func(processID string) {
		event := NewProcessRemovedEvent("", processID)
		updater.ApplyEvent(event)
	})

	return vm
}

// Refresh refreshes the territory from the current cluster state.
// This performs a full rebuild of the territory.
func (vm *ViewModel) Refresh() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Read cluster state
	snapshot, err := vm.reader.ReadClusterState()
	if err != nil {
		return fmt.Errorf("failed to read cluster state: %w", err)
	}

	// Detect processes
	processes, err := vm.detector.DetectProcesses()
	if err != nil {
		return fmt.Errorf("failed to detect processes: %w", err)
	}

	// Build territory with processes
	vm.territory = vm.mapper.BuildWorldWithProcesses(snapshot, processes)
	vm.updater = NewUpdater(vm.territory)
	vm.detector.SetWorld(vm.territory)

	return nil
}

// GetWorld returns the current World.
// Call Refresh() first to ensure the territory is up-to-date.
func (vm *ViewModel) GetWorld() *World {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.territory
}

// StartDetection starts background detection of process changes.
func (vm *ViewModel) StartDetection() error {
	return vm.detector.Start()
}

// StopDetection stops background detection.
func (vm *ViewModel) StopDetection() {
	vm.detector.Stop()
}

// Print prints the territory to the given writer.
func (vm *ViewModel) Print(w io.Writer) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	printer := NewPrinter(w)
	printer.PrintWorld(vm.territory)
}

// PrintSummary prints a summary to the given writer.
func (vm *ViewModel) PrintSummary(w io.Writer) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	printer := NewPrinter(w)
	printer.PrintSummary(vm.territory)
}

// PrintCompact prints a compact one-line summary.
func (vm *ViewModel) PrintCompact(w io.Writer) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	printer := NewPrinter(w)
	printer.PrintCompact(vm.territory)
}

// GetSummary returns a Summary of the territory.
func (vm *ViewModel) GetSummary() Summary {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.territory.GetSummary()
}

// LandCount returns the number of Land in the territory.
func (vm *ViewModel) LandCount() int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.territory.LandCount()
}

// ProcessCount returns the total number of processes.
func (vm *ViewModel) ProcessCount() int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.territory.TotalProcessCount()
}

// OnChange sets a callback for territory changes.
func (vm *ViewModel) OnChange(callback func(event Event)) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.updater.SetOnChange(callback)
}

// GetReader returns the underlying Reader (for advanced usage).
func (vm *ViewModel) GetReader() *Reader {
	return vm.reader
}

// GetDetector returns the underlying Detector (for advanced usage).
func (vm *ViewModel) GetDetector() *Detector {
	return vm.detector
}

// GetUpdater returns the underlying Updater (for advanced usage).
func (vm *ViewModel) GetUpdater() *Updater {
	return vm.updater
}

// QuickView creates a ViewModel, refreshes it, and returns the territory.
// This is a convenience function for one-shot usage.
func QuickView(ns *server.Server) (*World, error) {
	vm := New(ns)
	if err := vm.Refresh(); err != nil {
		return nil, err
	}
	return vm.GetWorld(), nil
}

// QuickPrint creates a ViewModel, refreshes it, and prints to the writer.
// This is a convenience function for one-shot CLI usage.
func QuickPrint(ns *server.Server, w io.Writer) error {
	vm := New(ns)
	if err := vm.Refresh(); err != nil {
		return err
	}
	vm.Print(w)
	return nil
}

// QuickSummary creates a ViewModel, refreshes it, and prints a summary.
// This is a convenience function for one-shot CLI usage.
func QuickSummary(ns *server.Server, w io.Writer) error {
	vm := New(ns)
	if err := vm.Refresh(); err != nil {
		return err
	}
	vm.PrintSummary(w)
	return nil
}

// ViewModelDancer is a Dancer that refreshes and prints the viewmodel on a schedule.
// It prints every N beats (at 90Hz, 450 beats = 5 seconds).
type ViewModelDancer struct {
	vm            *ViewModel
	printInterval uint64
	beatCount     uint64
}

// NewViewModelDancer creates a dancer that prints every printInterval beats.
func NewViewModelDancer(vm *ViewModel, printInterval uint64) *ViewModelDancer {
	return &ViewModelDancer{
		vm:            vm,
		printInterval: printInterval,
		beatCount:     0,
	}
}

// ID returns the dancer's identifier.
func (d *ViewModelDancer) ID() string {
	return "viewmodel-watcher"
}

// Beat represents a dance beat from WindWaker.
type Beat struct {
	Seq uint64 `json:"seq"`
	Ts  int64  `json:"ts"`
	Hz  int    `json:"hz"`
}

// Dance is called on each beat. It prints every printInterval beats.
func (d *ViewModelDancer) Dance(beat Beat) error {
	d.beatCount++

	if d.beatCount >= d.printInterval {
		d.beatCount = 0

		// Refresh and print
		if err := d.vm.Refresh(); err != nil {
			return fmt.Errorf("refresh failed: %w", err)
		}

		// Print with timestamp header
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("📊 Update at %s (beat #%d)\n",
			time.Now().Format("15:04:05"), beat.Seq)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		d.vm.PrintSummary(os.Stdout)
	}

	return nil
}
