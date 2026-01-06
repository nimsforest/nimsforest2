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
	case "pause":
		handlePause(cmdArgs)
	case "resume":
		handleResume(cmdArgs)
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
		printSources(status.Sources)
		fmt.Println()
		printTrees(status.Trees)
		fmt.Println()
		printTreeHouses(status.TreeHouses)
		fmt.Println()
		printNims(status.Nims)
	case "sources", "source", "src":
		printSources(status.Sources)
	case "trees", "tree":
		printTrees(status.Trees)
	case "treehouses", "treehouse", "th":
		printTreeHouses(status.TreeHouses)
	case "nims", "nim":
		printNims(status.Nims)
	default:
		fmt.Fprintf(os.Stderr, "Unknown type: %s (use: all, sources, trees, treehouses, nims)\n", what)
		os.Exit(1)
	}
}

func printSources(sources []runtime.SourceInfo) {
	fmt.Println("SOURCES:")
	if len(sources) == 0 {
		fmt.Println("  (none)")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tTYPE\tPUBLISHES\tDETAILS\tSTATUS")
	for _, s := range sources {
		status := "stopped"
		if s.Running {
			status = "running"
		}
		details := ""
		switch s.Type {
		case "http_webhook":
			details = s.Path
		case "http_poll":
			details = fmt.Sprintf("%s (%s)", s.URL, s.Interval)
		case "ceremony":
			details = s.Interval
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t[%s]\n",
			s.Name, s.Type, s.Publishes, details, status)
	}
	w.Flush()
}

func printTrees(trees []runtime.TreeInfo) {
	fmt.Println("TREES:")
	if len(trees) == 0 {
		fmt.Println("  (none)")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tWATCHES\tPUBLISHES\tSCRIPT\tSTATUS")
	for _, t := range trees {
		status := "stopped"
		if t.Running {
			status = "running"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t[%s]\n",
			t.Name, t.Watches, t.Publishes, t.Script, status)
	}
	w.Flush()
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
		fmt.Fprintln(os.Stderr, "Usage: forest add <source|tree|treehouse|nim> [options]")
		fmt.Fprintln(os.Stderr, "       forest add <source|tree|treehouse|nim> --config=<path>")
		os.Exit(1)
	}

	componentType := args[0]
	cmdArgs := args[1:]

	switch componentType {
	case "source", "src":
		handleAddSource(cmdArgs)
	case "tree":
		handleAddTree(cmdArgs)
	case "treehouse", "th":
		handleAddTreeHouse(cmdArgs)
	case "nim":
		handleAddNim(cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown component type: %s (use: source, tree, treehouse, nim)\n", componentType)
		os.Exit(1)
	}
}

func handleAddSource(args []string) {
	// Parse flags
	var name, sourceType, publishes, path, secret, url, method, interval, configPath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
		} else if strings.HasPrefix(arg, "--type=") {
			sourceType = strings.TrimPrefix(arg, "--type=")
		} else if strings.HasPrefix(arg, "--publishes=") {
			publishes = strings.TrimPrefix(arg, "--publishes=")
		} else if strings.HasPrefix(arg, "--path=") {
			path = strings.TrimPrefix(arg, "--path=")
		} else if strings.HasPrefix(arg, "--secret=") {
			secret = strings.TrimPrefix(arg, "--secret=")
		} else if strings.HasPrefix(arg, "--url=") {
			url = strings.TrimPrefix(arg, "--url=")
		} else if strings.HasPrefix(arg, "--method=") {
			method = strings.TrimPrefix(arg, "--method=")
		} else if strings.HasPrefix(arg, "--interval=") {
			interval = strings.TrimPrefix(arg, "--interval=")
		} else if strings.HasPrefix(arg, "--name=") {
			name = strings.TrimPrefix(arg, "--name=")
		} else if !strings.HasPrefix(arg, "-") && name == "" {
			name = arg
		}
	}

	client := runtime.NewClientFromEnv()

	// Load from config file if provided
	if configPath != "" {
		cfg, err := loadSourceConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		if name != "" {
			cfg.Name = name
		}
		if cfg.Name == "" {
			cfg.Name = strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
		}
		if err := client.AddSourceFromConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Added source '%s'\n", cfg.Name)
		return
	}

	// Validate required fields
	if name == "" {
		fmt.Fprintln(os.Stderr, "Error: name is required")
		printSourceHelp()
		os.Exit(1)
	}
	if sourceType == "" {
		fmt.Fprintln(os.Stderr, "Error: --type is required")
		printSourceHelp()
		os.Exit(1)
	}
	if publishes == "" {
		fmt.Fprintln(os.Stderr, "Error: --publishes is required")
		printSourceHelp()
		os.Exit(1)
	}

	opts := make(map[string]interface{})
	if path != "" {
		opts["path"] = path
	}
	if secret != "" {
		opts["secret"] = secret
	}
	if url != "" {
		opts["url"] = url
	}
	if method != "" {
		opts["method"] = method
	}
	if interval != "" {
		opts["interval"] = interval
	}

	if err := client.AddSource(name, sourceType, publishes, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Added source '%s'\n", name)
}

func printSourceHelp() {
	fmt.Fprintln(os.Stderr, "Usage: forest add source <name> --type=<type> --publishes=<subj> [options]")
	fmt.Fprintln(os.Stderr, "   or: forest add source --config=<path>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Types:")
	fmt.Fprintln(os.Stderr, "  http_webhook  HTTP webhook receiver")
	fmt.Fprintln(os.Stderr, "  http_poll     HTTP polling source")
	fmt.Fprintln(os.Stderr, "  ceremony      Interval-based trigger (counts WindWaker beats)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options for http_webhook:")
	fmt.Fprintln(os.Stderr, "  --path=/webhooks/name    HTTP endpoint path")
	fmt.Fprintln(os.Stderr, "  --secret=<secret>        Webhook signature secret")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options for http_poll:")
	fmt.Fprintln(os.Stderr, "  --url=<url>              URL to poll")
	fmt.Fprintln(os.Stderr, "  --interval=<duration>    Poll interval (e.g., 5m, 1h)")
	fmt.Fprintln(os.Stderr, "  --method=<method>        HTTP method (default: GET)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options for ceremony:")
	fmt.Fprintln(os.Stderr, "  --interval=<duration>    Trigger interval (e.g., 30s, 5m, 1h)")
}

func handleAddTree(args []string) {
	// Parse flags
	var name, watches, publishes, script, configPath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
		} else if strings.HasPrefix(arg, "--watches=") {
			watches = strings.TrimPrefix(arg, "--watches=")
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
		cfg, err := loadTreeConfig(configPath)
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
		if err := client.AddTreeFromConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Added tree '%s'\n", cfg.Name)
		return
	}

	// Validate required fields
	if name == "" {
		fmt.Fprintln(os.Stderr, "Error: name is required")
		fmt.Fprintln(os.Stderr, "Usage: forest add tree <name> --watches=<subj> --publishes=<subj> --script=<path>")
		fmt.Fprintln(os.Stderr, "   or: forest add tree --config=<path>")
		os.Exit(1)
	}
	if watches == "" || publishes == "" || script == "" {
		fmt.Fprintln(os.Stderr, "Error: --watches, --publishes, and --script are required")
		fmt.Fprintln(os.Stderr, "Usage: forest add tree <name> --watches=<subj> --publishes=<subj> --script=<path>")
		os.Exit(1)
	}

	if err := client.AddTree(name, watches, publishes, script); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Added tree '%s'\n", name)
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
		fmt.Fprintln(os.Stderr, "Usage: forest remove <source|tree|treehouse|nim> <name>")
		os.Exit(1)
	}

	componentType := args[0]
	name := args[1]

	client := runtime.NewClientFromEnv()

	switch componentType {
	case "source", "src":
		if err := client.RemoveSource(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Removed source '%s'\n", name)
	case "tree":
		if err := client.RemoveTree(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Removed tree '%s'\n", name)
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
		fmt.Fprintf(os.Stderr, "Unknown component type: %s (use: source, tree, treehouse, nim)\n", componentType)
		os.Exit(1)
	}
}

// =============================================================================
// Pause/Resume Commands (for sources)
// =============================================================================

func handlePause(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: forest pause source <name>")
		os.Exit(1)
	}

	componentType := args[0]
	name := args[1]

	client := runtime.NewClientFromEnv()

	switch componentType {
	case "source", "src":
		if err := client.PauseSource(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Paused source '%s'\n", name)
	default:
		fmt.Fprintf(os.Stderr, "Cannot pause %s (only sources can be paused)\n", componentType)
		os.Exit(1)
	}
}

func handleResume(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: forest resume source <name>")
		os.Exit(1)
	}

	componentType := args[0]
	name := args[1]

	client := runtime.NewClientFromEnv()

	switch componentType {
	case "source", "src":
		if err := client.ResumeSource(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Resumed source '%s'\n", name)
	default:
		fmt.Fprintf(os.Stderr, "Cannot resume %s (only sources can be resumed)\n", componentType)
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

// loadTreeConfig loads a Tree configuration from a YAML file.
func loadTreeConfig(path string) (runtime.TreeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runtime.TreeConfig{}, err
	}

	var cfg runtime.TreeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return runtime.TreeConfig{}, err
	}
	return cfg, nil
}

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

// loadSourceConfig loads a Source configuration from a YAML file.
func loadSourceConfig(path string) (runtime.SourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runtime.SourceConfig{}, err
	}

	var cfg runtime.SourceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return runtime.SourceConfig{}, err
	}
	return cfg, nil
}

func printClientHelp() {
	fmt.Print(`
CLI Commands (talk to running daemon):

  forest list [sources|trees|treehouses|nims|all]  List running components
  forest status [--json]                           Show daemon status
  forest add source <name> ...                     Add a source (External→River)
  forest add tree <name> ...                       Add a tree (River→Wind)
  forest add treehouse <name> ...                  Add a treehouse (Wind→Wind)
  forest add nim <name> ...                        Add a nim (Wind→Wind, AI)
  forest remove source <name>                      Remove a source
  forest remove tree <name>                        Remove a tree
  forest remove treehouse <name>                   Remove a treehouse
  forest remove nim <name>                         Remove a nim
  forest pause source <name>                       Pause a source
  forest resume source <name>                      Resume a source
  forest reload                                    Reload configuration from disk

Add Source Examples (feeds external data into River):
  forest add source stripe-webhook \
    --type=http_webhook \
    --path=/webhooks/stripe \
    --publishes=river.stripe.webhook \
    --secret=${STRIPE_WEBHOOK_SECRET}

  forest add source crm-contacts \
    --type=http_poll \
    --url=https://api.crm.com/contacts \
    --publishes=river.crm.contacts \
    --interval=5m

  forest add source heartbeat \
    --type=ceremony \
    --interval=30s \
    --publishes=river.system.heartbeat

  forest add source --config=./source.yaml

Add Tree Examples (parses external data from River):
  forest add tree stripe --watches=river.stripe.webhook --publishes=payment.completed --script=./parse_stripe.lua
  forest add tree --config=./tree.yaml

Add TreeHouse Examples (transforms internal Leaves):
  forest add treehouse scoring --subscribes=contact.created --publishes=lead.scored --script=./scoring.lua
  forest add treehouse --config=./treehouse.yaml

Add Nim Examples (AI-powered processing):
  forest add nim qualify --subscribes=lead.scored --publishes=lead.qualified --prompt=./qualify.md
  forest add nim --config=./nim.yaml

Environment:
  NIMSFOREST_API           API address (default: 127.0.0.1:8080)
  NIMSFOREST_WEBHOOK_ADDR  Webhook server address (default: 127.0.0.1:8081)
`)
}
