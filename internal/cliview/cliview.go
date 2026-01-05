// Package cliview provides a terminal/CLI view for the NimsForest cluster state.
//
// This package implements the View in the MVVM pattern, consuming data from
// the viewmodel package and presenting it to the terminal. It includes a
// Dancer implementation that registers with WindWaker to provide periodic
// updates.
package cliview

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/yourusername/nimsforest/internal/viewmodel"
	"github.com/yourusername/nimsforest/internal/windwaker"
)

// CLIView is a terminal view for the cluster state.
// It implements the Dancer interface to receive periodic updates.
type CLIView struct {
	vm            *viewmodel.ViewModel
	writer        io.Writer
	printInterval uint64 // Print every N beats
	beatCount     uint64
}

// New creates a new CLIView with the given viewmodel.
// printInterval is the number of beats between prints (at 90Hz, 450 = 5 seconds).
func New(vm *viewmodel.ViewModel, printInterval uint64) *CLIView {
	return &CLIView{
		vm:            vm,
		writer:        os.Stdout,
		printInterval: printInterval,
		beatCount:     0,
	}
}

// NewWithWriter creates a CLIView with a custom writer.
func NewWithWriter(vm *viewmodel.ViewModel, printInterval uint64, w io.Writer) *CLIView {
	return &CLIView{
		vm:            vm,
		writer:        w,
		printInterval: printInterval,
		beatCount:     0,
	}
}

// ID returns the dancer's identifier.
func (v *CLIView) ID() string {
	return "cliview"
}

// Dance is called on each beat from WindWaker.
// It prints the cluster state every printInterval beats.
func (v *CLIView) Dance(beat windwaker.Beat) error {
	v.beatCount++

	if v.beatCount >= v.printInterval {
		v.beatCount = 0

		// Refresh the viewmodel
		if err := v.vm.Refresh(); err != nil {
			return fmt.Errorf("refresh failed: %w", err)
		}

		// Print with timestamp header
		v.printHeader(beat.Seq)
		v.vm.PrintSummary(v.writer)
	}

	return nil
}

// printHeader prints a timestamped header for the output.
func (v *CLIView) printHeader(beatSeq uint64) {
	fmt.Fprintln(v.writer)
	fmt.Fprintln(v.writer, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(v.writer, "📊 Cluster State at %s\n", time.Now().Format("15:04:05"))
	fmt.Fprintln(v.writer, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// PrintOnce does a single refresh and print (useful for initial state).
func (v *CLIView) PrintOnce() error {
	if err := v.vm.Refresh(); err != nil {
		return err
	}
	v.printHeader(0)
	v.vm.PrintSummary(v.writer)
	return nil
}

// PrintFull does a single refresh and prints the full world view.
func (v *CLIView) PrintFull() error {
	if err := v.vm.Refresh(); err != nil {
		return err
	}
	v.printHeader(0)
	v.vm.Print(v.writer)
	return nil
}

// SetWriter changes the output writer.
func (v *CLIView) SetWriter(w io.Writer) {
	v.writer = w
}

// SetPrintInterval changes how often the view prints (in beats).
func (v *CLIView) SetPrintInterval(interval uint64) {
	v.printInterval = interval
}

// ViewModel returns the underlying viewmodel.
func (v *CLIView) ViewModel() *viewmodel.ViewModel {
	return v.vm
}
