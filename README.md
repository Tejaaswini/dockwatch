# Dockwatch — Docker Volumes & Disk Forensics TUI

A terminal user interface (TUI) for monitoring and managing Docker volumes with a focus on disk forensics and cleanup.

## Features

- **Volume Table**: View all Docker volumes with size, status, and metadata
- **Details Pane**: Inspect individual volume details and file previews
- **Prune Planning**: Mark volumes for deletion and see space savings
- **Real-time Data**: Connects directly to Docker daemon for live volume information

## Quick Start

```bash
# Initialize the Go module and download dependencies
go mod tidy

# Run the application
go run ./cmd/dockwatch

# Or build a binary
go build -o dockwatch ./cmd/dockwatch
./dockwatch
```

## Controls

- **↑/↓**: Move selection
- **Space**: Mark/unmark for prune
- **Enter**: Toggle details
- **P**: Open prune plan
- **Tab**: Cycle panes (Table → Details → Plan)
- **Q**: Quit


## Next Steps (TODO)

- Implement size scan via helper container (alpine) mounting volume read-only
- Add context switcher (docker contexts)
- Add JSON export (for CI)
- Add filters (only orphaned, sort by size)
- Add volume size monitoring over time
- Add alerts for volumes exceeding size thresholds

## Project Structure

```
dockwatch/
├── cmd/dockwatch/        # Main application entry point
├── internal/
│   ├── domain/           # Core data types (Volume struct)
│   ├── tui/              # Bubble Tea TUI implementation
│   ├── dockercli/        # Docker CLI integration
│   └── provider/         # Provider interface definitions
├── go.mod                # Go module definition
└── README.md             # This file
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
