package viewmodel

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// EventType represents the type of viewmodel event.
type EventType string

const (
	EventLandAdded      EventType = "land_added"
	EventLandRemoved    EventType = "land_removed"
	EventLandUpdated    EventType = "land_updated"
	EventProcessAdded   EventType = "process_added"
	EventProcessRemoved EventType = "process_removed"
	EventProcessUpdated EventType = "process_updated"
)

// Event represents a change to the viewmodel.
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	LandID    string      `json:"land_id,omitempty"`
	ProcessID string      `json:"process_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Updater applies events to a World for incremental updates.
type Updater struct {
	territory *World
	mu        sync.RWMutex

	// Event history for debugging/auditing
	eventHistory []Event
	maxHistory   int

	// Callbacks
	onChange func(event Event)
}

// NewUpdater creates a new Updater for the given World.
func NewUpdater(territory *World) *Updater {
	return &Updater{
		territory:    territory,
		eventHistory: make([]Event, 0),
		maxHistory:   100,
	}
}

// SetOnChange sets a callback that's called when an event is applied.
func (u *Updater) SetOnChange(callback func(event Event)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.onChange = callback
}

// ApplyEvent applies an event to the territory.
func (u *Updater) ApplyEvent(event Event) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	var err error
	switch event.Type {
	case EventLandAdded:
		err = u.applyLandAdded(event)
	case EventLandRemoved:
		err = u.applyLandRemoved(event)
	case EventLandUpdated:
		err = u.applyLandUpdated(event)
	case EventProcessAdded:
		err = u.applyProcessAdded(event)
	case EventProcessRemoved:
		err = u.applyProcessRemoved(event)
	case EventProcessUpdated:
		err = u.applyProcessUpdated(event)
	default:
		err = fmt.Errorf("unknown event type: %s", event.Type)
	}

	if err != nil {
		return err
	}

	// Record event in history
	u.recordEvent(event)

	// Call onChange callback
	if u.onChange != nil {
		u.onChange(event)
	}

	return nil
}

// applyLandAdded handles adding a new LandViewModel.
func (u *Updater) applyLandAdded(event Event) error {
	land, ok := event.Data.(*LandViewModel)
	if !ok {
		// Try to create from NodeInfo
		if nodeInfo, ok := event.Data.(NodeInfo); ok {
			mapper := NewMapper()
			land = mapper.nodeToLand(nodeInfo)
		} else {
			return fmt.Errorf("invalid data for land_added event")
		}
	}

	if land.ID == "" {
		land.ID = event.LandID
	}

	u.territory.AddLand(land)
	log.Printf("[Updater] Land added: %s", land.ID)
	return nil
}

// applyLandRemoved handles removing a LandViewModel.
func (u *Updater) applyLandRemoved(event Event) error {
	if event.LandID == "" {
		return fmt.Errorf("land_id required for land_removed event")
	}

	if !u.territory.RemoveLand(event.LandID) {
		return fmt.Errorf("land not found: %s", event.LandID)
	}

	log.Printf("[Updater] Land removed: %s", event.LandID)
	return nil
}

// applyLandUpdated handles updating a LandViewModel's properties.
func (u *Updater) applyLandUpdated(event Event) error {
	if event.LandID == "" {
		return fmt.Errorf("land_id required for land_updated event")
	}

	land := u.territory.GetLand(event.LandID)
	if land == nil {
		return fmt.Errorf("land not found: %s", event.LandID)
	}

	// Apply updates from event data
	if updates, ok := event.Data.(map[string]interface{}); ok {
		if ram, ok := updates["ram_total"].(uint64); ok {
			land.RAMTotal = ram
		}
		if cpu, ok := updates["cpu_cores"].(int); ok {
			land.CPUCores = cpu
		}
		if vram, ok := updates["gpu_vram"].(uint64); ok {
			land.GPUVram = vram
		}
		if tflops, ok := updates["gpu_tflops"].(float64); ok {
			land.GPUTflops = tflops
		}
	}

	land.LastSeen = time.Now()
	log.Printf("[Updater] Land updated: %s", event.LandID)
	return nil
}

// applyProcessAdded handles adding a process to a LandViewModel.
func (u *Updater) applyProcessAdded(event Event) error {
	if event.LandID == "" {
		return fmt.Errorf("land_id required for process_added event")
	}

	land := u.territory.GetLand(event.LandID)
	if land == nil {
		return fmt.Errorf("land not found: %s", event.LandID)
	}

	// Handle different process types
	switch proc := event.Data.(type) {
	case TreeViewModel:
		land.AddTree(proc)
		log.Printf("[Updater] Tree added to %s: %s", event.LandID, proc.Name)
	case TreehouseViewModel:
		land.AddTreehouse(proc)
		log.Printf("[Updater] Treehouse added to %s: %s", event.LandID, proc.Name)
	case NimViewModel:
		land.AddNim(proc)
		log.Printf("[Updater] Nim added to %s: %s", event.LandID, proc.Name)
	case DetectedProcess:
		// Convert DetectedProcess to appropriate type
		switch proc.Type {
		case ProcessTypeTree:
			tree := NewTreeViewModel(proc.ID, proc.Name, proc.RAMAllocated, proc.Subjects)
			land.AddTree(tree)
		case ProcessTypeTreehouse:
			th := NewTreehouseViewModel(proc.ID, proc.Name, proc.RAMAllocated, proc.ScriptPath)
			land.AddTreehouse(th)
		case ProcessTypeNim:
			nim := NewNimViewModel(proc.ID, proc.Name, proc.RAMAllocated, proc.Subjects, proc.AIEnabled)
			land.AddNim(nim)
		}
		log.Printf("[Updater] Process added to %s: %s (%s)", event.LandID, proc.Name, proc.Type)
	default:
		return fmt.Errorf("invalid process data type: %T", event.Data)
	}

	return nil
}

// applyProcessRemoved handles removing a process from a LandViewModel.
func (u *Updater) applyProcessRemoved(event Event) error {
	if event.ProcessID == "" {
		return fmt.Errorf("process_id required for process_removed event")
	}

	// If LandID is specified, look only in that land
	if event.LandID != "" {
		land := u.territory.GetLand(event.LandID)
		if land == nil {
			return fmt.Errorf("land not found: %s", event.LandID)
		}
		if !land.RemoveProcess(event.ProcessID) {
			return fmt.Errorf("process not found: %s", event.ProcessID)
		}
	} else {
		// Search all lands
		found := false
		for _, land := range u.territory.Lands() {
			if land.RemoveProcess(event.ProcessID) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("process not found: %s", event.ProcessID)
		}
	}

	log.Printf("[Updater] Process removed: %s", event.ProcessID)
	return nil
}

// applyProcessUpdated handles updating a process.
func (u *Updater) applyProcessUpdated(event Event) error {
	if event.ProcessID == "" {
		return fmt.Errorf("process_id required for process_updated event")
	}

	proc, land := u.territory.FindProcess(event.ProcessID)
	if proc == nil {
		return fmt.Errorf("process not found: %s", event.ProcessID)
	}

	// Apply updates
	if updates, ok := event.Data.(map[string]interface{}); ok {
		if ram, ok := updates["ram_allocated"].(uint64); ok {
			proc.RAMAllocated = ram
		}
		if name, ok := updates["name"].(string); ok {
			proc.Name = name
		}
	}

	log.Printf("[Updater] Process updated on %s: %s", land.ID, event.ProcessID)
	return nil
}

// recordEvent adds an event to the history.
func (u *Updater) recordEvent(event Event) {
	u.eventHistory = append(u.eventHistory, event)
	if len(u.eventHistory) > u.maxHistory {
		u.eventHistory = u.eventHistory[1:]
	}
}

// GetEventHistory returns recent events.
func (u *Updater) GetEventHistory() []Event {
	u.mu.RLock()
	defer u.mu.RUnlock()

	history := make([]Event, len(u.eventHistory))
	copy(history, u.eventHistory)
	return history
}

// NewLandAddedEvent creates a land_added event.
func NewLandAddedEvent(land *LandViewModel) Event {
	return Event{
		Type:      EventLandAdded,
		Timestamp: time.Now(),
		LandID:    land.ID,
		Data:      land,
	}
}

// NewLandRemovedEvent creates a land_removed event.
func NewLandRemovedEvent(landID string) Event {
	return Event{
		Type:      EventLandRemoved,
		Timestamp: time.Now(),
		LandID:    landID,
	}
}

// NewProcessAddedEvent creates a process_added event.
func NewProcessAddedEvent(landID string, proc interface{}) Event {
	var processID string
	switch p := proc.(type) {
	case TreeViewModel:
		processID = p.ID
	case TreehouseViewModel:
		processID = p.ID
	case NimViewModel:
		processID = p.ID
	case DetectedProcess:
		processID = p.ID
	}

	return Event{
		Type:      EventProcessAdded,
		Timestamp: time.Now(),
		LandID:    landID,
		ProcessID: processID,
		Data:      proc,
	}
}

// NewProcessRemovedEvent creates a process_removed event.
func NewProcessRemovedEvent(landID, processID string) Event {
	return Event{
		Type:      EventProcessRemoved,
		Timestamp: time.Now(),
		LandID:    landID,
		ProcessID: processID,
	}
}
