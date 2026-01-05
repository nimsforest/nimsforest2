package viewmodel

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Printer formats Territory data for output.
type Printer struct {
	writer io.Writer
}

// NewPrinter creates a new Printer that writes to the given writer.
func NewPrinter(w io.Writer) *Printer {
	return &Printer{writer: w}
}

// PrintSummary prints a summary of the territory.
// Output format:
//
//	Capacity:
//	  Land: 2 (ram: 32GB, cpu: 8 cores)
//	  Manaland: 1 (ram: 32GB, cpu: 8 cores, vram: 24GB)
//	  Total: 3 land (ram: 64GB, cpu: 16 cores, vram: 24GB)
//
//	Usage:
//	  Trees: 2 (ram: 8GB)
//	  Treehouses: 1 (ram: 1GB)
//	  Nims: 1 (ram: 2GB)
//	  Total: 4 processes (ram: 11GB, 17% of capacity)
func (p *Printer) PrintSummary(territory *TerritoryViewModel) {
	summary := territory.GetSummary()
	
	// Calculate regular land stats
	regularRAM := uint64(0)
	regularCPU := 0
	for _, land := range territory.Lands() {
		if !land.IsManaland() {
			regularRAM += land.RAMTotal
			regularCPU += land.CPUCores
		}
	}
	
	// Calculate manaland stats
	manaRAM := uint64(0)
	manaCPU := 0
	for _, land := range territory.Lands() {
		if land.IsManaland() {
			manaRAM += land.RAMTotal
			manaCPU += land.CPUCores
		}
	}
	
	fmt.Fprintln(p.writer, "Capacity:")
	if summary.LandCount > 0 {
		fmt.Fprintf(p.writer, "  Land: %d (ram: %s, cpu: %d cores)\n",
			summary.LandCount, FormatBytes(regularRAM), regularCPU)
	}
	if summary.ManalandCount > 0 {
		fmt.Fprintf(p.writer, "  Manaland: %d (ram: %s, cpu: %d cores, vram: %s)\n",
			summary.ManalandCount, FormatBytes(manaRAM), manaCPU, FormatBytes(summary.TotalGPUVram))
	}
	
	totalLand := summary.LandCount + summary.ManalandCount
	if summary.TotalGPUVram > 0 {
		fmt.Fprintf(p.writer, "  Total: %d land (ram: %s, cpu: %d cores, vram: %s)\n",
			totalLand, FormatBytes(summary.TotalRAM), summary.TotalCPUCores, FormatBytes(summary.TotalGPUVram))
	} else {
		fmt.Fprintf(p.writer, "  Total: %d land (ram: %s, cpu: %d cores)\n",
			totalLand, FormatBytes(summary.TotalRAM), summary.TotalCPUCores)
	}
	
	fmt.Fprintln(p.writer)
	fmt.Fprintln(p.writer, "Usage:")
	
	if summary.TreeCount > 0 {
		fmt.Fprintf(p.writer, "  Trees: %d (ram: %s)\n",
			summary.TreeCount, FormatBytes(summary.TreeRAM))
	} else {
		fmt.Fprintln(p.writer, "  Trees: 0")
	}
	
	if summary.TreehouseCount > 0 {
		fmt.Fprintf(p.writer, "  Treehouses: %d (ram: %s)\n",
			summary.TreehouseCount, FormatBytes(summary.TreehouseRAM))
	} else {
		fmt.Fprintln(p.writer, "  Treehouses: 0")
	}
	
	if summary.NimCount > 0 {
		fmt.Fprintf(p.writer, "  Nims: %d (ram: %s)\n",
			summary.NimCount, FormatBytes(summary.NimRAM))
	} else {
		fmt.Fprintln(p.writer, "  Nims: 0")
	}
	
	totalProcesses := summary.TreeCount + summary.TreehouseCount + summary.NimCount
	if totalProcesses > 0 {
		fmt.Fprintf(p.writer, "  Total: %d processes (ram: %s, %.0f%% of capacity)\n",
			totalProcesses, FormatBytes(summary.TotalRAMAllocated), summary.Occupancy)
	} else {
		fmt.Fprintln(p.writer, "  Total: 0 processes")
	}
}

// PrintTerritory prints the full territory with all land and processes.
// Output format:
//
//	Territory: 3 land
//
//	Land: node-abc (ram: 16GB, cpu: 4, occupancy: 38%)
//	  Trees:
//	    - payment-processor (ram: 4GB)
//	  Treehouses:
//	    - scoring (ram: 1GB)
//	  Nims:
//	    - qualify (ram: 2GB)
//
//	Land: node-xyz (ram: 16GB, cpu: 4, occupancy: 25%)
//	  Trees:
//	    - router (ram: 4GB)
//	  Treehouses: (none)
//	  Nims: (none)
func (p *Printer) PrintTerritory(territory *TerritoryViewModel) {
	totalLand := territory.LandCount()
	fmt.Fprintf(p.writer, "Territory: %d land\n", totalLand)
	
	lands := territory.Lands()
	for i, land := range lands {
		fmt.Fprintln(p.writer)
		p.printLand(land)
		
		// Add extra newline between lands (but not after the last one)
		if i < len(lands)-1 {
			// Space is already added by the blank line between lands
		}
	}
}

// printLand prints a single LandViewModel with its processes.
func (p *Printer) printLand(land *LandViewModel) {
	// Determine land type
	landType := "Land"
	if land.IsManaland() {
		landType = "Manaland"
	}
	
	// Print land header
	if land.IsManaland() {
		fmt.Fprintf(p.writer, "%s: %s (ram: %s, cpu: %d, gpu: %s vram, occupancy: %.0f%%)\n",
			landType, land.ID, FormatBytes(land.RAMTotal), land.CPUCores,
			FormatBytes(land.GPUVram), land.Occupancy())
	} else {
		fmt.Fprintf(p.writer, "%s: %s (ram: %s, cpu: %d, occupancy: %.0f%%)\n",
			landType, land.ID, FormatBytes(land.RAMTotal), land.CPUCores, land.Occupancy())
	}
	
	// Print Trees
	fmt.Fprint(p.writer, "  Trees:")
	if len(land.Trees) == 0 {
		fmt.Fprintln(p.writer, " (none)")
	} else {
		fmt.Fprintln(p.writer)
		for _, tree := range land.Trees {
			fmt.Fprintf(p.writer, "    - %s (ram: %s)\n", tree.Name, FormatBytes(tree.RAMAllocated))
		}
	}
	
	// Print Treehouses
	fmt.Fprint(p.writer, "  Treehouses:")
	if len(land.Treehouses) == 0 {
		fmt.Fprintln(p.writer, " (none)")
	} else {
		fmt.Fprintln(p.writer)
		for _, th := range land.Treehouses {
			fmt.Fprintf(p.writer, "    - %s (ram: %s)\n", th.Name, FormatBytes(th.RAMAllocated))
		}
	}
	
	// Print Nims
	fmt.Fprint(p.writer, "  Nims:")
	if len(land.Nims) == 0 {
		fmt.Fprintln(p.writer, " (none)")
	} else {
		fmt.Fprintln(p.writer)
		for _, nim := range land.Nims {
			aiTag := ""
			if nim.AIEnabled {
				aiTag = " [AI]"
			}
			fmt.Fprintf(p.writer, "    - %s (ram: %s)%s\n", nim.Name, FormatBytes(nim.RAMAllocated), aiTag)
		}
	}
}

// PrintCompact prints a compact single-line summary.
func (p *Printer) PrintCompact(territory *TerritoryViewModel) {
	summary := territory.GetSummary()
	totalLand := summary.LandCount + summary.ManalandCount
	
	fmt.Fprintf(p.writer, "%d land | %d trees, %d treehouses, %d nims | %s/%s RAM (%.0f%%)\n",
		totalLand, summary.TreeCount, summary.TreehouseCount, summary.NimCount,
		FormatBytes(summary.TotalRAMAllocated), FormatBytes(summary.TotalRAM), summary.Occupancy)
}

// PrintJSON prints the territory as JSON (for machine consumption).
func (p *Printer) PrintJSON(territory *TerritoryViewModel) error {
	// For now, just use the built-in JSON marshaling
	// In a production implementation, you'd use encoding/json
	lands := territory.Lands()
	
	fmt.Fprintln(p.writer, "{")
	fmt.Fprintf(p.writer, "  \"land_count\": %d,\n", len(lands))
	fmt.Fprintln(p.writer, "  \"lands\": [")
	
	for i, land := range lands {
		p.printLandJSON(land, "    ")
		if i < len(lands)-1 {
			fmt.Fprintln(p.writer, ",")
		} else {
			fmt.Fprintln(p.writer)
		}
	}
	
	fmt.Fprintln(p.writer, "  ]")
	fmt.Fprintln(p.writer, "}")
	return nil
}

// printLandJSON prints a land as JSON.
func (p *Printer) printLandJSON(land *LandViewModel, indent string) {
	fmt.Fprintf(p.writer, "%s{\n", indent)
	fmt.Fprintf(p.writer, "%s  \"id\": \"%s\",\n", indent, land.ID)
	fmt.Fprintf(p.writer, "%s  \"hostname\": \"%s\",\n", indent, land.Hostname)
	fmt.Fprintf(p.writer, "%s  \"ram_total\": %d,\n", indent, land.RAMTotal)
	fmt.Fprintf(p.writer, "%s  \"cpu_cores\": %d,\n", indent, land.CPUCores)
	fmt.Fprintf(p.writer, "%s  \"gpu_vram\": %d,\n", indent, land.GPUVram)
	fmt.Fprintf(p.writer, "%s  \"occupancy\": %.2f,\n", indent, land.Occupancy())
	fmt.Fprintf(p.writer, "%s  \"tree_count\": %d,\n", indent, len(land.Trees))
	fmt.Fprintf(p.writer, "%s  \"treehouse_count\": %d,\n", indent, len(land.Treehouses))
	fmt.Fprintf(p.writer, "%s  \"nim_count\": %d\n", indent, len(land.Nims))
	fmt.Fprintf(p.writer, "%s}", indent)
}

// FormatDuration formats a duration in a human-readable way.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// Box draws a simple box around text for emphasis.
func (p *Printer) Box(title string, content string) {
	width := len(title) + 4
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if len(line)+4 > width {
			width = len(line) + 4
		}
	}
	
	// Top border
	fmt.Fprintf(p.writer, "┌%s┐\n", strings.Repeat("─", width-2))
	
	// Title
	padding := (width - 2 - len(title)) / 2
	fmt.Fprintf(p.writer, "│%s%s%s│\n",
		strings.Repeat(" ", padding),
		title,
		strings.Repeat(" ", width-2-padding-len(title)))
	
	// Separator
	fmt.Fprintf(p.writer, "├%s┤\n", strings.Repeat("─", width-2))
	
	// Content
	for _, line := range lines {
		fmt.Fprintf(p.writer, "│ %-*s │\n", width-4, line)
	}
	
	// Bottom border
	fmt.Fprintf(p.writer, "└%s┘\n", strings.Repeat("─", width-2))
}
