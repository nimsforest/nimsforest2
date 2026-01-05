package viewmodel

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

// Detector monitors the cluster for process changes.
// It detects Trees, Treehouses, and Nims by analyzing subscription patterns.
type Detector struct {
	reader    *Reader
	territory *Territory
	mu        sync.RWMutex
	
	// Known process patterns for identification
	treePatterns      []string
	treehousePatterns []string
	nimPatterns       []string
	
	// Process registry - tracks what we've detected
	knownProcesses map[string]DetectedProcess
	
	// Event callbacks
	onProcessAdded   func(proc DetectedProcess)
	onProcessRemoved func(processID string)
	
	// Polling configuration
	pollInterval time.Duration
	stopCh       chan struct{}
	running      bool
}

// NewDetector creates a new Detector for the given Reader.
func NewDetector(reader *Reader) *Detector {
	return &Detector{
		reader:         reader,
		knownProcesses: make(map[string]DetectedProcess),
		treePatterns: []string{
			"river.>",
			"river.*",
			"river.stripe.>",
			"river.general.>",
		},
		treehousePatterns: []string{
			"contact.>",
			"contact.created",
			"lead.scored",
		},
		nimPatterns: []string{
			"payment.>",
			"payment.completed",
			"payment.failed",
			"lead.qualified",
			"followup.>",
			"data.>",
			"status.>",
			"notification.>",
		},
		pollInterval: 5 * time.Second,
		stopCh:       make(chan struct{}),
	}
}

// SetTerritory sets the territory to update when processes are detected.
func (d *Detector) SetTerritory(territory *Territory) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.territory = territory
}

// SetOnProcessAdded sets a callback for when a process is detected.
func (d *Detector) SetOnProcessAdded(callback func(proc DetectedProcess)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onProcessAdded = callback
}

// SetOnProcessRemoved sets a callback for when a process is removed.
func (d *Detector) SetOnProcessRemoved(callback func(processID string)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onProcessRemoved = callback
}

// DetectProcesses performs a one-time scan of subscriptions and detects processes.
func (d *Detector) DetectProcesses() ([]DetectedProcess, error) {
	// Get current subscriptions
	subs, err := d.reader.GetSubscriptions()
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	// Get JetStream consumers
	_, consumers, err := d.reader.GetJetStreamInfo()
	if err != nil {
		log.Printf("[Detector] Warning: failed to get JetStream info: %v", err)
	}

	// Get connection info to understand what's connected
	connz, err := d.reader.GetConnz()
	if err != nil {
		log.Printf("[Detector] Warning: failed to get connection info: %v", err)
	}

	var processes []DetectedProcess

	// Analyze subscriptions to detect processes
	processMap := make(map[string]DetectedProcess)
	
	for _, sub := range subs {
		proc := d.analyzeSubscription(sub)
		if proc != nil {
			// Use a combination of subject and queue as key to avoid duplicates
			key := fmt.Sprintf("%s:%s", sub.Subject, sub.Queue)
			if existing, ok := processMap[key]; ok {
				// Merge subjects if same process
				existing.Subjects = append(existing.Subjects, proc.Subjects...)
				processMap[key] = existing
			} else {
				processMap[key] = *proc
			}
		}
	}

	// Analyze JetStream consumers
	for _, consumer := range consumers {
		proc := d.analyzeConsumer(consumer)
		if proc != nil {
			key := fmt.Sprintf("consumer:%s:%s", consumer.StreamName, consumer.Name)
			processMap[key] = *proc
		}
	}

	// Analyze connections if available
	if connz != nil {
		for _, conn := range connz.Conns {
			if conn.Name != "" && strings.Contains(conn.Name, "nimsforest") {
				// This is an internal connection, likely a component
				for _, sub := range conn.Subs {
					proc := d.analyzeSubject(sub)
					if proc != nil {
						key := fmt.Sprintf("conn:%s:%s", conn.Name, sub)
						if _, ok := processMap[key]; !ok {
							processMap[key] = *proc
						}
					}
				}
			}
		}
	}

	for _, proc := range processMap {
		processes = append(processes, proc)
	}

	return processes, nil
}

// analyzeSubscription analyzes a subscription to detect process type.
func (d *Detector) analyzeSubscription(sub SubscriptionInfo) *DetectedProcess {
	if sub.Subject == "" {
		return nil
	}

	// Skip internal NATS subjects
	if strings.HasPrefix(sub.Subject, "_") ||
		strings.HasPrefix(sub.Subject, "$") {
		return nil
	}

	procType := d.inferType(sub.Subject)
	name := InferProcessName(sub.Subject)
	
	proc := &DetectedProcess{
		ID:           fmt.Sprintf("%s-%s", procType, name),
		Name:         name,
		Type:         procType,
		RAMAllocated: 256 * 1024 * 1024, // Default 256MB
		Subjects:     []string{sub.Subject},
	}

	return proc
}

// analyzeConsumer analyzes a JetStream consumer to detect process type.
func (d *Detector) analyzeConsumer(consumer ConsumerInfo) *DetectedProcess {
	if consumer.FilterSubject == "" {
		return nil
	}

	procType := d.inferType(consumer.FilterSubject)
	name := consumer.Name
	if name == "" {
		name = InferProcessName(consumer.FilterSubject)
	}

	proc := &DetectedProcess{
		ID:           fmt.Sprintf("%s-%s", procType, name),
		Name:         name,
		Type:         procType,
		RAMAllocated: 256 * 1024 * 1024,
		Subjects:     []string{consumer.FilterSubject},
	}

	return proc
}

// analyzeSubject analyzes a subject string to create a process.
func (d *Detector) analyzeSubject(subject string) *DetectedProcess {
	if subject == "" {
		return nil
	}

	// Skip internal NATS subjects
	if strings.HasPrefix(subject, "_") ||
		strings.HasPrefix(subject, "$") {
		return nil
	}

	procType := d.inferType(subject)
	name := InferProcessName(subject)

	proc := &DetectedProcess{
		ID:           fmt.Sprintf("%s-%s", procType, name),
		Name:         name,
		Type:         procType,
		RAMAllocated: 256 * 1024 * 1024,
		Subjects:     []string{subject},
	}

	return proc
}

// inferType infers the process type from a subject pattern.
func (d *Detector) inferType(subject string) ProcessType {
	subject = strings.ToLower(subject)

	// Check tree patterns
	for _, pattern := range d.treePatterns {
		if matchesPattern(subject, pattern) {
			return ProcessTypeTree
		}
	}

	// Check treehouse patterns
	for _, pattern := range d.treehousePatterns {
		if matchesPattern(subject, pattern) {
			return ProcessTypeTreehouse
		}
	}

	// Check nim patterns
	for _, pattern := range d.nimPatterns {
		if matchesPattern(subject, pattern) {
			return ProcessTypeNim
		}
	}

	// Use the generic inference as fallback
	return InferProcessType(subject)
}

// matchesPattern checks if a subject matches a NATS pattern.
func matchesPattern(subject, pattern string) bool {
	// Simple pattern matching for NATS wildcards
	if pattern == subject {
		return true
	}

	// Handle ">" wildcard (matches any number of tokens)
	if strings.HasSuffix(pattern, ".>") {
		prefix := strings.TrimSuffix(pattern, ".>")
		return strings.HasPrefix(subject, prefix+".")
	}

	// Handle "*" wildcard (matches single token)
	if strings.Contains(pattern, "*") {
		patternParts := strings.Split(pattern, ".")
		subjectParts := strings.Split(subject, ".")
		
		if len(patternParts) != len(subjectParts) {
			return false
		}
		
		for i, pp := range patternParts {
			if pp != "*" && pp != subjectParts[i] {
				return false
			}
		}
		return true
	}

	return false
}

// Start begins polling for subscription changes.
func (d *Detector) Start() error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("detector already running")
	}
	d.running = true
	d.mu.Unlock()

	go d.pollLoop()
	log.Printf("[Detector] Started with poll interval %v", d.pollInterval)
	return nil
}

// Stop stops the detector polling.
func (d *Detector) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	d.mu.Unlock()

	close(d.stopCh)
	log.Printf("[Detector] Stopped")
}

// pollLoop continuously polls for changes.
func (d *Detector) pollLoop() {
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.checkForChanges()
		}
	}
}

// checkForChanges checks for new or removed processes.
func (d *Detector) checkForChanges() {
	processes, err := d.DetectProcesses()
	if err != nil {
		log.Printf("[Detector] Error detecting processes: %v", err)
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Create a map of current processes
	currentProcesses := make(map[string]DetectedProcess)
	for _, proc := range processes {
		currentProcesses[proc.ID] = proc
	}

	// Check for new processes
	for id, proc := range currentProcesses {
		if _, known := d.knownProcesses[id]; !known {
			d.knownProcesses[id] = proc
			if d.onProcessAdded != nil {
				d.onProcessAdded(proc)
			}
			log.Printf("[Detector] New process detected: %s (%s)", proc.Name, proc.Type)
		}
	}

	// Check for removed processes
	for id := range d.knownProcesses {
		if _, exists := currentProcesses[id]; !exists {
			delete(d.knownProcesses, id)
			if d.onProcessRemoved != nil {
				d.onProcessRemoved(id)
			}
			log.Printf("[Detector] Process removed: %s", id)
		}
	}
}

// GetKnownProcesses returns all currently known processes.
func (d *Detector) GetKnownProcesses() []DetectedProcess {
	d.mu.RLock()
	defer d.mu.RUnlock()

	processes := make([]DetectedProcess, 0, len(d.knownProcesses))
	for _, proc := range d.knownProcesses {
		processes = append(processes, proc)
	}
	return processes
}

// DetectorFromServer creates a Detector directly from a NATS server.
func DetectorFromServer(ns *server.Server) *Detector {
	reader := NewReader(ns)
	return NewDetector(reader)
}
