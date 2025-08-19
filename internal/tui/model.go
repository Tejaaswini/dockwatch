package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"dockwatch/internal/dockercli"
	"dockwatch/internal/domain"
	"dockwatch/internal/provider"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	borderStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

type pane int

const (
	paneTable pane = iota
	paneDetails
	panePlan
)

type model struct {
	ready  bool
	active pane

	vols   []domain.Volume
	table  table.Model
	marked map[int]bool // row index -> marked

	showDetails bool

	// Provider management
	provider provider.Provider
	ctx      context.Context
}

func New() model {
	// Start with Docker provider by default
	dockerProv, err := getDockerProvider()
	if err != nil {
		// If Docker fails, create a model with error state
		fmt.Printf("Failed to connect to Docker: %v\n", err)
		fmt.Printf("Make sure Docker is running and accessible\n")
		return model{
			active:   paneTable,
			vols:     []domain.Volume{},
			table:    table.New(),
			marked:   map[int]bool{},
			provider: nil,
			ctx:      context.Background(),
		}
	}

	// Build columns
	cols := []table.Column{
		{Title: "Name", Width: 28},
		{Title: "Size", Width: 10},
		{Title: "Attached", Width: 18},
		{Title: "Project", Width: 14},
		{Title: "Status", Width: 8},
	}

	// Load Docker data
	vols, err := dockerProv.ListVolumes(context.Background())
	if err != nil {
		fmt.Printf("Failed to load volumes: %v\n", err)
		vols = []domain.Volume{}
	}

	rows := make([]table.Row, 0, len(vols))
	for _, v := range vols {
		attached := "<none>"
		if len(v.Attached) > 0 {
			attached = strings.Join(v.Attached, ",")
		}
		status := "ACTIVE"
		if v.Orphan {
			status = "ORPHAN"
		}
		rows = append(rows, table.Row{v.Name, v.SizeHuman(), attached, v.Project, status})
	}

	t := table.New(table.WithColumns(cols), table.WithRows(rows), table.WithFocused(true))
	t.KeyMap.LineUp.SetKeys("up")
	t.KeyMap.LineDown.SetKeys("down")

	return model{
		active:   paneTable,
		vols:     vols,
		table:    t,
		marked:   map[int]bool{},
		provider: dockerProv,
		ctx:      context.Background(),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			if m.provider != nil {
				m.provider.Close()
			}
			return m, tea.Quit
		case "tab":
			m.active = (m.active + 1) % 3
		case "enter":
			m.showDetails = !m.showDetails
		case "p":
			m.active = panePlan
		case " ":
			idx := m.table.Cursor()
			m.marked[idx] = !m.marked[idx]
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := titleStyle.Render("Docker Volumes — Real Data")

	// Add status info
	statusInfo := fmt.Sprintf("Volumes: %d", len(m.vols))
	header = header + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(statusInfo)

	// Top table with markers
	rendered := m.renderTable()

	// Details / Plan panes
	lower := ""
	switch m.active {
	case paneDetails:
		lower = m.renderDetails()
	case panePlan:
		lower = m.renderPlan()
	default:
		lower = helpText()
	}

	return header + "\n" + rendered + "\n" + lower
}

func (m model) renderTable() string {
	// decorate selected row if marked
	rows := m.table.Rows()
	for i := range rows {
		mark := " "
		if m.marked[i] {
			mark = "✓"
		}
		// prepend checkbox to name
		rows[i][0] = fmt.Sprintf("[%s] %s", mark, rows[i][0])
	}
	// render inside border
	return borderStyle.Width(80).Render(m.table.View())
}

func (m model) renderDetails() string {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.vols) {
		return ""
	}
	v := m.vols[idx]
	attached := "<none>"
	if len(v.Attached) > 0 {
		attached = strings.Join(v.Attached, ", ")
	}
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "Details: %s (%s)\n", v.Name, v.SizeHuman())
	fmt.Fprintf(sb, "Driver: %s\n", v.Driver)
	fmt.Fprintf(sb, "Project: %s\n", ifEmpty(v.Project, "<none>"))
	fmt.Fprintf(sb, "Status: %s\n", tern(v.Orphan, "ORPHAN", "ACTIVE"))
	fmt.Fprintf(sb, "Attached: %s\n", attached)
	fmt.Fprintf(sb, "\nReal Docker volume data\n")

	return borderStyle.Width(80).Render(sb.String())
}

func (m model) renderPlan() string {
	total := int64(0)
	lines := make([]string, 0)
	for i, v := range m.vols {
		if m.marked[i] {
			lines = append(lines, fmt.Sprintf("  ✓ %s (%s)", v.Name, v.SizeHuman()))
			if v.SizeBytes > 0 {
				total += v.SizeBytes
			}
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "  <none selected>")
	}
	human := humanBytes(total)
	body := "Prune Plan:\n" + strings.Join(lines, "\n") + "\n\nTotal space to reclaim: " + human + "\n\n[A] Apply prune   [C] Cancel   [Q] Quit"
	return borderStyle.Width(80).Render(body)
}

func helpText() string {
	return borderStyle.Width(80).Render("[↑/↓] Move  [Space] Mark  [Enter] Details  [P] Plan  [Tab] Switch  [Q] Quit")
}

func humanBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const kb = 1024
	const mb = kb * 1024
	const gb = mb * 1024
	fb := float64(b)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.2f GB", fb/gb)
	case b >= mb:
		return fmt.Sprintf("%.2f MB", fb/mb)
	case b >= kb:
		return fmt.Sprintf("%.2f KB", fb/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func ifEmpty(s, repl string) string {
	if s == "" {
		return repl
	}
	return s
}

func tern[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

// getDockerProvider creates a Docker provider, returns error if Docker is not available
func getDockerProvider() (provider.Provider, error) {
	dockerProv, err := dockercli.NewDockerProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker provider: %w", err)
	}
	return dockerProv, nil
}
