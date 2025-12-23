# NimsForest - Quick Start Guide

**For developers who want to get started immediately**

---

## 🚀 One-Command Setup

```bash
git clone <repository>
cd nimsforest
make dev
```

That's it! This command:
- ✅ Installs NATS server (if needed)
- ✅ Downloads Go dependencies
- ✅ Creates directory structure
- ✅ Verifies environment
- ✅ Starts NATS with JetStream
- ✅ Runs integration tests

---

## 📋 Essential Commands

### Daily Development

```bash
make start              # Start NATS server
make stop               # Stop NATS server
make status             # Check if NATS is running
make test               # Run tests
make build              # Build the application
make run                # Build and run
```

### Code Quality

```bash
make fmt                # Format code
make vet                # Run go vet
make lint               # Run linter (if installed)
make check              # Run all quality checks
```

### Testing

```bash
make test               # Unit tests
make test-integration   # Integration tests (auto-starts NATS)
make test-coverage      # Tests with coverage report
```

### Cleanup

```bash
make clean              # Remove build artifacts
make clean-data         # Remove NATS data (prompts for confirmation)
make restart            # Restart NATS
```

---

## 🆘 Common Issues

### "NATS won't start"
```bash
make status             # Check current status
make stop               # Stop any stuck processes
make start              # Start fresh
```

### "Tests failing"
```bash
make restart            # Restart NATS
make test-integration   # Run tests again
```

### "Need fresh environment"
```bash
make clean-all          # Clean everything
make setup              # Setup from scratch
make start              # Start NATS
```

---

## 📖 Get Help

```bash
make help               # Show all available commands
```

---

## 🔍 Quick Reference

| Task | Command |
|------|---------|
| First-time setup | `make dev` |
| Start working | `make start` |
| Run tests | `make test` |
| Check status | `make status` |
| Build app | `make build` |
| Format code | `make fmt` |
| Stop NATS | `make stop` |
| See all commands | `make help` |

---

## 📚 Documentation

- **Full setup guide**: `README.md`
- **Task breakdown**: `TASK_BREAKDOWN.md`

---

**That's everything you need to get started!** 🎉

For more details, run `make help` or read `README.md`.
