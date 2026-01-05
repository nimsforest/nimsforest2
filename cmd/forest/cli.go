package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/yourusername/nimsforest/pkg/runtime"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// CLI Client Commands
// These commands talk to a running nimsforest daemon via HTTP API
// =============================================================================

// runClientCommand handles CLI commands that talk to the daemon.
func runClientCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No command specified")
		printClientHelp()
		os.Exit(1)
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "list", "ls":
		handleList(cmdArgs)
	case "status":
		handleStatus(cmdArgs)
	case "add":
		handleAdd(cmdArgs)
	case "remove", "rm":
		handleRemove(cmdArgs)
	case "reload":
		handleReload(cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printClientHelp()
		os.Exit(1)
	}
}

// =============================================================================
// List Command
// =============================================================================

func handleList(args []string) {
	client := runtime.NewClientFromEnv()

	// Check what to list
	what := "all"
	if len(args) > 0 {
		what = args[0]
	}

	status, err := client.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Is the nimsforest daemon running?")
		os.Exit(1)
	}

	switch what {
	case "all":
		printTreeHouses(status.TreeHouses)
		fmt.Println()
		printNims(status.Nims)
	case "treehouses", "treehouse", "th":
		printTreeHouses(status.TreeHouses)
	case "nims", "nim":
		printNims(status.Nims)
	default:
		fmt.Fprintf(os.Stderr, "Unknown type: %s (use: all, treehouses, nims)\n", what)
		os.Exit(1)
	}
}

func printTreeHouses(treehouses []runtime.ComponentInfo) {
	fmt.Println("TREEHOUSES:")
	if len(treehouses) == 0 {
		fmt.Println("  (none)")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tSUBSCRIBES\tPUBLISHES\tSCRIPT\tSTATUS")
	for _, th := range treehouses {
		status := "stopped"
		if th.Running {
			status = "running"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t[%s]\n",
			th.Name, th.Subscribes, th.Publishes, th.Script, status)
	}
	w.Flush()
}

func printNims(nims []runtime.ComponentInfo) {
	fmt.Println("NIMS:")
	if len(nims) == 0 {
		fmt.Println("  (none)")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tSUBSCRIBES\tPUBLISHES\tPROMPT\tSTATUS")
	for _, nim := range nims {
		status := "stopped"
		if nim.Running {
			status = "running"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t[%s]\n",
			nim.Name, nim.Subscribes, nim.Publishes, nim.Prompt, status)
	}
	w.Flush()
}

// =============================================================================
// Status Command
// =============================================================================

func handleStatus(args []string) {
	client := runtime.NewClientFromEnv()

	status, err := client.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Is the nimsforest daemon running?")
		os.Exit(1)
	}

	// Check for --json flag
	for _, arg := range args {
		if arg == "--json" || arg == "-j" {
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(data))
			return
		}
	}

	// Human-readable output
	runningStatus := "🔴 stopped"
	if status.Running {
		runningStatus = "🟢 running"
	}

	fmt.Printf("NimsForest Status: %s\n", runningStatus)
	fmt.Printf("Config: %s\n", status.ConfigPath)
	fmt.Printf("TreeHouses: %d\n", len(status.TreeHouses))
	fmt.Printf("Nims: %d\n", len(status.Nims))
}

// =============================================================================
// Add Command
// =============================================================================

func handleAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: forest add <treehouse|nim> [options]")
		fmt.Fprintln(os.Stderr, "       forest add <treehouse|nim> --config=<path>")
		os.Exit(1)
	}

	componentType := args[0]
	cmdArgs := args[1:]

	switch componentType {
	case "treehouse", "th":
		handleAddTreeHouse(cmdArgs)
	case "nim":
		handleAddNim(cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown component type: %s (use: treehouse, nim)\n", componentType)
		os.Exit(1)
	}
}

func handleAddTreeHouse(args []string) {
	// Parse flags
	var name, subscribes, publishes, script, configPath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
		} else if strings.HasPrefix(arg, "--subscribes=") {
			subscribes = strings.TrimPrefix(arg, "--subscribes=")
		} else if strings.HasPrefix(arg, "--publishes=") {
			publishes = strings.TrimPrefix(arg, "--publishes=")
		} else if strings.HasPrefix(arg, "--script=") {
			script = strings.TrimPrefix(arg, "--script=")
		} else if strings.HasPrefix(arg, "--name=") {
			name = strings.TrimPrefix(arg, "--name=")
		} else if !strings.HasPrefix(arg, "-") && name == "" {
			name = arg
		}
	}

	client := runtime.NewClientFromEnv()

	// Load from config file if provided
	if configPath != "" {
		cfg, err := loadTreeHouseConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		if name != "" {
			cfg.Name = name // Allow overriding name
		}
		if cfg.Name == "" {
			// Use filename as name
			cfg.Name = strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
		}
		if err := client.AddTreeHouseFromConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Added treehouse '%s'\n", cfg.Name)
		return
	}

	// Validate required fields
	if name == "" {
		fmt.Fprintln(os.Stderr, "Error: name is required")
		fmt.Fprintln(os.Stderr, "Usage: forest add treehouse <name> --subscribes=<subj> --publishes=<subj> --script=<path>")
		fmt.Fprintln(os.Stderr, "   or: forest add treehouse --config=<path>")
		os.Exit(1)
	}
	if subscribes == "" || publishes == "" || script == "" {
		fmt.Fprintln(os.Stderr, "Error: --subscribes, --publishes, and --script are required")
		fmt.Fprintln(os.Stderr, "Usage: forest add treehouse <name> --subscribes=<subj> --publishes=<subj> --script=<path>")
		os.Exit(1)
	}

	if err := client.AddTreeHouse(name, subscribes, publishes, script); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Added treehouse '%s'\n", name)
}

func handleAddNim(args []string) {
	// Parse flags
	var name, subscribes, publishes, prompt, configPath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
		} else if strings.HasPrefix(arg, "--subscribes=") {
			subscribes = strings.TrimPrefix(arg, "--subscribes=")
		} else if strings.HasPrefix(arg, "--publishes=") {
			publishes = strings.TrimPrefix(arg, "--publishes=")
		} else if strings.HasPrefix(arg, "--prompt=") {
			prompt = strings.TrimPrefix(arg, "--prompt=")
		} else if strings.HasPrefix(arg, "--name=") {
			name = strings.TrimPrefix(arg, "--name=")
		} else if !strings.HasPrefix(arg, "-") && name == "" {
			name = arg
		}
	}

	client := runtime.NewClientFromEnv()

	// Load from config file if provided
	if configPath != "" {
		cfg, err := loadNimConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		if name != "" {
			cfg.Name = name // Allow overriding name
		}
		if cfg.Name == "" {
			// Use filename as name
			cfg.Name = strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
		}
		if err := client.AddNimFromConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Added nim '%s'\n", cfg.Name)
		return
	}

	// Validate required fields
	if name == "" {
		fmt.Fprintln(os.Stderr, "Error: name is required")
		fmt.Fprintln(os.Stderr, "Usage: forest add nim <name> --subscribes=<subj> --publishes=<subj> --prompt=<path>")
		fmt.Fprintln(os.Stderr, "   or: forest add nim --config=<path>")
		os.Exit(1)
	}
	if subscribes == "" || publishes == "" || prompt == "" {
		fmt.Fprintln(os.Stderr, "Error: --subscribes, --publishes, and --prompt are required")
		fmt.Fprintln(os.Stderr, "Usage: forest add nim <name> --subscribes=<subj> --publishes=<subj> --prompt=<path>")
		os.Exit(1)
	}

	if err := client.AddNim(name, subscribes, publishes, prompt); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Added nim '%s'\n", name)
}

// =============================================================================
// Remove Command
// =============================================================================

func handleRemove(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: forest remove <treehouse|nim> <name>")
		os.Exit(1)
	}

	componentType := args[0]
	name := args[1]

	client := runtime.NewClientFromEnv()

	switch componentType {
	case "treehouse", "th":
		if err := client.RemoveTreeHouse(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Removed treehouse '%s'\n", name)
	case "nim":
		if err := client.RemoveNim(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Removed nim '%s'\n", name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown component type: %s (use: treehouse, nim)\n", componentType)
		os.Exit(1)
	}
}

// =============================================================================
// Reload Command
// =============================================================================

func handleReload(args []string) {
	client := runtime.NewClientFromEnv()

	if err := client.Reload(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Configuration reloaded")
}

// =============================================================================
// Helpers
// =============================================================================

// loadTreeHouseConfig loads a TreeHouse configuration from a YAML file.
func loadTreeHouseConfig(path string) (runtime.TreeHouseConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runtime.TreeHouseConfig{}, err
	}

	var cfg runtime.TreeHouseConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return runtime.TreeHouseConfig{}, err
	}
	return cfg, nil
}

// loadNimConfig loads a Nim configuration from a YAML file.
func loadNimConfig(path string) (runtime.NimConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runtime.NimConfig{}, err
	}

	var cfg runtime.NimConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return runtime.NimConfig{}, err
	}
	return cfg, nil
}

func printClientHelp() {
	fmt.Print(`
CLI Commands (talk to running daemon):

  forest list [treehouses|nims|all]    List running components
  forest status [--json]               Show daemon status
  forest add treehouse <name> ...      Add a treehouse
  forest add nim <name> ...            Add a nim
  forest remove treehouse <name>       Remove a treehouse
  forest remove nim <name>             Remove a nim
  forest reload                        Reload configuration from disk

Add TreeHouse Examples:
  forest add treehouse scoring --subscribes=contact.created --publishes=lead.scored --script=./scoring.lua
  forest add treehouse --config=./treehouse.yaml

Add Nim Examples:
  forest add nim qualify --subscribes=lead.scored --publishes=lead.qualified --prompt=./qualify.md
  forest add nim --config=./nim.yaml

Environment:
  NIMSFOREST_API    API address (default: 127.0.0.1:8080)
`)
}
