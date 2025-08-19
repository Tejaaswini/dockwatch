package domain

import (
	"fmt"
	"time"
)

// Volume represents a Docker volume (or mock) with basic metadata.
type Volume struct {
	Name      string
	Driver    string
	SizeBytes int64    // may be -1 if unknown
	Attached  []string // container names
	Project   string   // from labels (compose)
	Orphan    bool
	LastSeen  time.Time // optional
}

func (v Volume) SizeHuman() string {
	b := float64(v.SizeBytes)
	if v.SizeBytes < 0 {
		return "?"
	}
	const kb = 1024
	const mb = kb * 1024
	const gb = mb * 1024
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", b/gb)
	case b >= mb:
		return fmt.Sprintf("%.1f MB", b/mb)
	case b >= kb:
		return fmt.Sprintf("%.1f KB", b/kb)
	default:
		return fmt.Sprintf("%d B", int64(b))
	}
}
