package viewerdancer

import (
	"context"
	"sync"

	viewer "github.com/nimsforest/nimsforestviewer"
	"github.com/yourusername/nimsforest/internal/viewmodel"
	"github.com/yourusername/nimsforest/internal/windwaker"
)

// ViewerDancer implements windwaker.Dancer to push viewmodel updates
// to nimsforestviewer targets (Smart TV, web, etc).
type ViewerDancer struct {
	viewer        *viewer.Viewer
	vm            *viewmodel.ViewModel
	updateEvery   uint64 // Update every N beats (0 = every beat)
	beatCount     uint64
	mu            sync.Mutex
	lastState     *viewer.ViewState
	onlyOnChange  bool // Only update if state actually changed
}

// Option configures a ViewerDancer.
type Option func(*ViewerDancer)

// WithUpdateInterval sets how often to push updates (every N beats).
// At 90Hz, 90 = once per second, 45 = twice per second.
func WithUpdateInterval(beats uint64) Option {
	return func(d *ViewerDancer) {
		d.updateEvery = beats
	}
}

// WithOnlyOnChange only pushes updates when state actually changes.
// Useful for Smart TV to avoid "connecting" messages.
func WithOnlyOnChange(enabled bool) Option {
	return func(d *ViewerDancer) {
		d.onlyOnChange = enabled
	}
}

// New creates a new ViewerDancer.
func New(vm *viewmodel.ViewModel, v *viewer.Viewer, opts ...Option) *ViewerDancer {
	d := &ViewerDancer{
		viewer:       v,
		vm:           vm,
		updateEvery:  90, // Default: once per second at 90Hz
		onlyOnChange: true,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// ID implements windwaker.Dancer.
func (d *ViewerDancer) ID() string {
	return "viewer-dancer"
}

// Dance implements windwaker.Dancer.
// Called on each windwaker beat to potentially update the viewer.
func (d *ViewerDancer) Dance(beat windwaker.Beat) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.beatCount++

	// Check if it's time to update
	if d.updateEvery > 0 && d.beatCount < d.updateEvery {
		return nil
	}
	d.beatCount = 0

	// Refresh the viewmodel
	if err := d.vm.Refresh(); err != nil {
		return err
	}

	// Convert to ViewState
	world := d.vm.GetWorld()
	state := ConvertToViewState(world)

	// Check if state changed (if onlyOnChange is enabled)
	if d.onlyOnChange && d.lastState != nil {
		if statesEqual(d.lastState, state) {
			return nil // No change, skip update
		}
	}
	d.lastState = state

	// Push to viewer
	d.viewer.SetStateProvider(viewer.NewStaticStateProvider(state))
	return d.viewer.Update()
}

// ForceUpdate triggers an immediate update regardless of interval.
func (d *ViewerDancer) ForceUpdate() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.vm.Refresh(); err != nil {
		return err
	}

	world := d.vm.GetWorld()
	state := ConvertToViewState(world)
	d.lastState = state

	d.viewer.SetStateProvider(viewer.NewStaticStateProvider(state))
	return d.viewer.Update()
}

// Viewer returns the underlying viewer for adding targets.
func (d *ViewerDancer) Viewer() *viewer.Viewer {
	return d.viewer
}

// statesEqual compares two ViewStates for equality (shallow comparison).
func statesEqual(a, b *viewer.ViewState) bool {
	if a == nil || b == nil {
		return a == b
	}
	if len(a.Lands) != len(b.Lands) {
		return false
	}
	// Compare summary
	if a.Summary.TotalLands != b.Summary.TotalLands ||
		a.Summary.TotalTrees != b.Summary.TotalTrees ||
		a.Summary.TotalNims != b.Summary.TotalNims ||
		a.Summary.AllocatedRAM != b.Summary.AllocatedRAM {
		return false
	}
	// Compare lands (basic check - count processes)
	for i := range a.Lands {
		if len(a.Lands[i].Trees) != len(b.Lands[i].Trees) ||
			len(a.Lands[i].Nims) != len(b.Lands[i].Nims) ||
			len(a.Lands[i].Treehouses) != len(b.Lands[i].Treehouses) {
			return false
		}
	}
	return true
}

// NewWithTargets creates a ViewerDancer with common targets pre-configured.
// Discovers Smart TVs and optionally starts a web server.
func NewWithTargets(ctx context.Context, vm *viewmodel.ViewModel, webAddr string, opts ...Option) (*ViewerDancer, error) {
	v := viewer.New()

	// Add web target if address specified
	if webAddr != "" {
		webTarget, err := viewer.NewWebTarget(webAddr)
		if err != nil {
			return nil, err
		}
		v.AddTarget(webTarget)
	}

	return New(vm, v, opts...), nil
}
