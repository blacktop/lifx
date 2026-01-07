package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/blacktop/lifx/internal/backend"
	"github.com/blacktop/lifx/internal/config"
	"github.com/blacktop/lifx/internal/models"
	"github.com/blacktop/lifx/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type statusLevel int

const (
	statusInfo statusLevel = iota
	statusWarn
	statusErr
)

type statusMessage struct {
	text  string
	level statusLevel
}

type listItem struct {
	isGroup bool
	group   *models.Group
	light   *models.Light
}

type colorPreset struct {
	Name  string
	Color models.Color
}

type dataMsg struct {
	state *backend.State
}

type errMsg struct {
	err error
}

type statusClearMsg struct{}

type autoRefreshMsg struct{}

type Model struct {
	backend        backend.Backend
	backendWarning string
	autoRefresh    time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	locations []*models.Location
	groups    []*models.Group
	scenes    []*models.Scene

	items         []listItem
	selectedIndex int
	scrollOffset  int

	showPanel bool

	searchMode  bool
	searchInput textinput.Model
	searchQuery string

	showScenes  bool
	scenesModal ScenesModel

	colorPresets []colorPreset
	presetIndex  int

	loading bool
	spinner spinner.Model

	width  int
	height int

	status statusMessage
}

// NewModel creates a new TUI model.
func NewModel(ctx context.Context, be backend.Backend, warn string, autoRefresh time.Duration) Model {
	ctx, cancel := context.WithCancel(ctx)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(styles.ColorPrimary)

	search := textinput.New()
	search.Placeholder = "Search lights..."
	search.CharLimit = 64

	return Model{
		backend:        be,
		backendWarning: warn,
		autoRefresh:    autoRefresh,
		ctx:            ctx,
		cancel:         cancel,
		spinner:        sp,
		loading:        true,
		showPanel:      true,
		searchInput:    search,
		scenesModal:    NewScenesModel(),
		colorPresets:   loadPresets(),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.SetWindowTitle("LIFX"),
		m.spinner.Tick,
		m.refreshCmd(),
	}
	if m.autoRefresh > 0 {
		cmds = append(cmds, m.autoRefreshTickCmd())
	}
	if m.backendWarning != "" {
		cmds = append(cmds, m.setStatusCmd(m.backendWarning, statusWarn))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureVisible()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case dataMsg:
		m.loading = false
		m.locations = msg.state.Locations
		m.groups = msg.state.Groups
		m.scenes = msg.state.Scenes
		m.scenesModal.SetScenes(m.scenes, m.locations)
		m.rebuildItems()
		return m, nil

	case errMsg:
		m.loading = false
		return m, m.setStatusCmd(msg.err.Error(), statusErr)

	case statusClearMsg:
		m.status = statusMessage{}
		return m, nil

	case autoRefreshMsg:
		// Trigger a background refresh and schedule the next tick
		var cmds []tea.Cmd
		cmds = append(cmds, m.refreshCmd())
		if m.autoRefresh > 0 {
			cmds = append(cmds, m.autoRefreshTickCmd())
		}
		return m, tea.Batch(cmds...)

	case sceneActivateMsg:
		m.showScenes = false
		return m, m.activateSceneCmd(msg.sceneID)

	case closeScenesMsg:
		m.showScenes = false
		return m, nil
	}

	if m.showScenes {
		updated, cmd := m.scenesModal.Update(msg)
		m.scenesModal = updated
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.String() {
			case "enter":
				m.searchQuery = strings.TrimSpace(m.searchInput.Value())
				m.searchMode = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.rebuildItems()
				return m, nil
			case "esc":
				m.searchMode = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				return m, nil
			}
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.cancel()
			return m, tea.Quit

		case "tab":
			m.showPanel = !m.showPanel
			return m, nil

		case "/":
			m.searchMode = true
			m.searchInput.Focus()
			return m, nil

		case "esc":
			if m.searchQuery != "" {
				m.searchQuery = ""
				m.rebuildItems()
			}
			return m, nil

		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.refreshCmd())

		case "s":
			m.showScenes = true
			return m, nil

		case "j", "down":
			m.moveDown()
			return m, nil

		case "k", "up":
			m.moveUp()
			return m, nil

		case "h", "left":
			return m, m.adjustBrightnessCmd(-5)

		case "l", "right":
			return m, m.adjustBrightnessCmd(5)

		case "w":
			return m, m.adjustKelvinCmd(-250)

		case "c":
			return m, m.adjustKelvinCmd(250)

		case "p":
			return m, m.applyPresetCmd(1)

		case "P":
			return m, m.applyPresetCmd(-1)

		case " ":
			return m, m.togglePowerCmd()

		case "a":
			return m, m.setGroupPowerCmd(true)

		case "x":
			return m, m.setGroupPowerCmd(false)

		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			pct := brightnessFromKey(msg.String())
			return m, m.setBrightnessCmd(pct)
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	if m.searchMode {
		b.WriteString(styles.StyleSearch.Render("/ ") + m.searchInput.View())
		b.WriteString("\n")
	} else if m.searchQuery != "" {
		b.WriteString(styles.StyleSearch.Render("/ " + m.searchQuery + " "))
		b.WriteString(styles.StyleMuted.Render("(esc to clear)"))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	contentWidth := m.width
	panelWidth := 0
	showPanel := m.showPanel && m.width >= 80
	if showPanel {
		panelWidth = m.width * 30 / 100
		if panelWidth < 30 {
			panelWidth = 30
		}
		if panelWidth > 46 {
			panelWidth = 46
		}
		contentWidth = m.width - panelWidth - 3
	}

	content := m.renderList(contentWidth)
	contentHeight := m.contentHeight()
	contentStyle := lipgloss.NewStyle().Height(contentHeight).MaxHeight(contentHeight)

	if showPanel {
		panel := m.renderPanel(panelWidth)
		contentStyle = contentStyle.Width(contentWidth)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, contentStyle.Render(content), "  ", panel))
	} else {
		b.WriteString(contentStyle.Render(content))
	}

	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	if m.showScenes {
		modal := m.scenesModal.View()
		overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		return overlay
	}

	return b.String()
}

func (m *Model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		state, err := m.backend.Refresh(m.ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return dataMsg{state: state}
	}
}

func (m *Model) autoRefreshTickCmd() tea.Cmd {
	return tea.Tick(m.autoRefresh, func(time.Time) tea.Msg {
		return autoRefreshMsg{}
	})
}

func (m *Model) setStatusCmd(text string, level statusLevel) tea.Cmd {
	m.status = statusMessage{text: text, level: level}
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return statusClearMsg{} })
}

func (m *Model) activateSceneCmd(sceneID string) tea.Cmd {
	return func() tea.Msg {
		if err := m.backend.ActivateScene(m.ctx, sceneID); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}

func (m *Model) togglePowerCmd() tea.Cmd {
	lights := m.selectedLights()
	if len(lights) == 0 {
		return nil
	}
	var cmds []tea.Cmd
	for _, light := range lights {
		light.On = !light.On
		id := light.ID
		on := light.On
		idCopy := id
		onCopy := on
		cmds = append(cmds, func() tea.Msg {
			if err := m.backend.SetLightPower(m.ctx, idCopy, onCopy); err != nil {
				return errMsg{err: err}
			}
			return nil
		})
	}
	m.refreshGroupStates()
	return tea.Batch(cmds...)
}

func (m *Model) setGroupPowerCmd(on bool) tea.Cmd {
	item := m.selectedItem()
	if item == nil || !item.isGroup {
		return nil
	}
	var cmds []tea.Cmd
	for _, light := range item.group.Lights {
		light.On = on
		id := light.ID
		idCopy := id
		onCopy := on
		cmds = append(cmds, func() tea.Msg {
			if err := m.backend.SetLightPower(m.ctx, idCopy, onCopy); err != nil {
				return errMsg{err: err}
			}
			return nil
		})
	}
	m.refreshGroupStates()
	return tea.Batch(cmds...)
}

func (m *Model) adjustBrightnessCmd(delta int) tea.Cmd {
	lights := m.selectedLights()
	if len(lights) == 0 {
		return nil
	}
	var cmds []tea.Cmd
	for _, light := range lights {
		light.SetBrightnessPct(light.BrightnessPct() + delta)
		color := light.Color
		id := light.ID
		colorCopy := color
		idCopy := id
		cmds = append(cmds, func() tea.Msg {
			if err := m.backend.SetLightColor(m.ctx, idCopy, colorCopy); err != nil {
				return errMsg{err: err}
			}
			return nil
		})
	}
	m.refreshGroupStates()
	return tea.Batch(cmds...)
}

func (m *Model) setBrightnessCmd(pct int) tea.Cmd {
	lights := m.selectedLights()
	if len(lights) == 0 {
		return nil
	}
	var cmds []tea.Cmd
	for _, light := range lights {
		light.SetBrightnessPct(pct)
		color := light.Color
		id := light.ID
		colorCopy := color
		idCopy := id
		cmds = append(cmds, func() tea.Msg {
			if err := m.backend.SetLightColor(m.ctx, idCopy, colorCopy); err != nil {
				return errMsg{err: err}
			}
			return nil
		})
	}
	m.refreshGroupStates()
	return tea.Batch(cmds...)
}

func (m *Model) adjustKelvinCmd(delta int) tea.Cmd {
	lights := m.selectedLights()
	if len(lights) == 0 {
		return nil
	}
	var cmds []tea.Cmd
	for _, light := range lights {
		light.Color.Kelvin += delta
		light.Color.Saturation = 0
		light.Color = light.Color.Clamp()
		id := light.ID
		color := light.Color
		colorCopy := color
		idCopy := id
		cmds = append(cmds, func() tea.Msg {
			if err := m.backend.SetLightColor(m.ctx, idCopy, colorCopy); err != nil {
				return errMsg{err: err}
			}
			return nil
		})
	}
	return tea.Batch(cmds...)
}

func (m *Model) applyPresetCmd(step int) tea.Cmd {
	if len(m.colorPresets) == 0 {
		return nil
	}
	m.presetIndex += step
	if m.presetIndex < 0 {
		m.presetIndex = len(m.colorPresets) - 1
	}
	if m.presetIndex >= len(m.colorPresets) {
		m.presetIndex = 0
	}

	lights := m.selectedLights()
	if len(lights) == 0 {
		return nil
	}
	preset := m.colorPresets[m.presetIndex]
	var cmds []tea.Cmd
	for _, light := range lights {
		color := preset.Color
		color.Brightness = light.BrightnessPct()
		if color.Kelvin == 0 {
			color.Kelvin = light.Color.Kelvin
		}
		light.Color = color.Clamp()
		id := light.ID
		colorCopy := light.Color
		idCopy := id
		cmds = append(cmds, func() tea.Msg {
			if err := m.backend.SetLightColor(m.ctx, idCopy, colorCopy); err != nil {
				return errMsg{err: err}
			}
			return nil
		})
	}
	return tea.Batch(cmds...)
}

func (m *Model) rebuildItems() {
	m.items = nil
	groups := append([]*models.Group(nil), m.groups...)
	sort.Slice(groups, func(i, j int) bool { return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name) })

	query := strings.ToLower(strings.TrimSpace(m.searchQuery))
	for _, group := range groups {
		var lights []*models.Light
		for _, light := range group.Lights {
			if query == "" || strings.Contains(strings.ToLower(light.Name), query) {
				lights = append(lights, light)
			}
		}
		if len(lights) == 0 {
			continue
		}
		m.items = append(m.items, listItem{isGroup: true, group: group})
		sort.Slice(lights, func(i, j int) bool { return strings.ToLower(lights[i].Name) < strings.ToLower(lights[j].Name) })
		for _, light := range lights {
			m.items = append(m.items, listItem{light: light})
		}
	}

	if m.selectedIndex >= len(m.items) {
		m.selectedIndex = len(m.items) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	m.ensureVisible()
}

func (m *Model) ensureVisible() {
	visible := m.visibleLines()
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}
	if m.selectedIndex >= m.scrollOffset+visible {
		m.scrollOffset = m.selectedIndex - visible + 1
	}
	maxScroll := len(m.items) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *Model) visibleLines() int {
	contentHeight := m.contentHeight() - 2
	if contentHeight < 2 {
		contentHeight = 2
	}
	return contentHeight
}

func (m *Model) contentHeight() int {
	contentHeight := m.height - 5
	if m.searchMode || m.searchQuery != "" {
		contentHeight -= 1
	}
	if contentHeight < 3 {
		contentHeight = 3
	}
	return contentHeight
}

func (m *Model) moveDown() {
	if m.selectedIndex < len(m.items)-1 {
		m.selectedIndex++
	}
	m.ensureVisible()
}

func (m *Model) moveUp() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
	m.ensureVisible()
}

func (m *Model) selectedItem() *listItem {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.items) {
		return nil
	}
	return &m.items[m.selectedIndex]
}

func (m *Model) selectedLights() []*models.Light {
	item := m.selectedItem()
	if item == nil {
		return nil
	}
	if item.isGroup && item.group != nil {
		return item.group.Lights
	}
	if item.light != nil {
		return []*models.Light{item.light}
	}
	return nil
}

func (m *Model) refreshGroupStates() {
	for _, group := range m.groups {
		group.UpdateState()
	}
}

func (m *Model) renderHeader() string {
	label := styles.StyleHeader.Render(" LIFX ")
	backendBadge := styles.StyleHeaderBadge.Render(" " + m.backend.Name() + " ")
	var status string
	if m.loading {
		status = lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(" ⟳ Loading...")
	} else {
		status = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(" ● Connected")
	}
	return label + backendBadge + status
}

func (m *Model) renderList(width int) string {
	var b strings.Builder
	visible := m.visibleLines()
	end := m.scrollOffset + visible
	if end > len(m.items) {
		end = len(m.items)
	}

	if m.scrollOffset > 0 {
		b.WriteString(styles.StyleDim.Render(fmt.Sprintf("  ↑ %d more", m.scrollOffset)))
		b.WriteString("\n")
	}

	for idx := m.scrollOffset; idx < end; idx++ {
		item := m.items[idx]
		selected := idx == m.selectedIndex
		if item.isGroup {
			if idx > m.scrollOffset {
				b.WriteString("\n")
			}
			b.WriteString(m.renderGroupHeader(item.group, selected))
			b.WriteString("\n")
			continue
		}
		b.WriteString(m.renderLightRow(item.light, selected, width))
		b.WriteString("\n")
	}

	if end < len(m.items) {
		b.WriteString(styles.StyleDim.Render(fmt.Sprintf("  ↓ %d more", len(m.items)-end)))
		b.WriteString("\n")
	}

	if len(m.items) == 0 {
		if m.loading {
			b.WriteString("  " + m.spinner.View() + " Searching for lights...")
		} else {
			b.WriteString(styles.StyleDim.Render("  No lights found"))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *Model) renderGroupHeader(group *models.Group, selected bool) string {
	if group == nil {
		return ""
	}

	// Cursor indicator
	cursor := styles.StyleMuted.Render("  ")
	if selected {
		cursor = styles.StyleSelected.Render("> ")
	}

	// Group name
	name := group.Name
	if group.LocationName != "" {
		name = fmt.Sprintf("%s — %s", group.Name, group.LocationName)
	}

	// Count lights on and calculate average brightness
	lightsOn := 0
	totalBrightness := 0
	for _, light := range group.Lights {
		if light.On {
			lightsOn++
			totalBrightness += light.BrightnessPct()
		}
	}

	// Build summary
	summary := fmt.Sprintf("(%d/%d on", lightsOn, len(group.Lights))
	if lightsOn > 0 {
		avgBrightness := totalBrightness / lightsOn
		summary += fmt.Sprintf(" • %d%%", avgBrightness)
	}
	summary += ")"

	nameStyle := styles.StyleGroupTitle
	if selected {
		nameStyle = styles.StyleSelected
	}

	return fmt.Sprintf("%s%s %s", cursor, nameStyle.Render(name), styles.StyleMuted.Render(summary))
}

func (m *Model) renderLightRow(light *models.Light, selected bool, width int) string {
	if light == nil {
		return ""
	}

	// Cursor indicator
	cursor := styles.StyleMuted.Render("  ")
	if selected {
		cursor = styles.StyleSelected.Render("> ")
	}

	// Power icon
	icon := styles.StyleLightOff.Render("○")
	if light.On {
		icon = styles.StyleLightOn.Render("●")
	}

	// Calculate layout dynamically
	fixedParts := 15 // cursor(2) + icon(1) + spaces + bar + pct
	availableForName := width - fixedParts

	barWidth := 12
	if width > 80 {
		barWidth = 16
	}

	nameWidth := availableForName - barWidth - 8
	if nameWidth < 12 {
		nameWidth = 12
	}
	if nameWidth > 35 {
		nameWidth = 35
	}

	// Name styling
	nameStyle := styles.StyleLightNameDim
	if light.On {
		nameStyle = styles.StyleLightName
	}
	if selected {
		nameStyle = styles.StyleSelected
	}
	name := nameStyle.Render(truncateName(light.Name, nameWidth))

	// Brightness bar with gradient
	bar := renderGradientBar(light.BrightnessPct(), light.On, barWidth)

	// Percentage
	pct := styles.StyleMuted.Render(fmt.Sprintf("%3d%%", light.BrightnessPct()))

	// Color indicator
	colorInd := ""
	if light.On {
		hex := light.Color.Hex()
		colorInd = lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(" ◆")
	}

	return fmt.Sprintf("%s%s %s  %s %s%s", cursor, icon, name, bar, pct, colorInd)
}

func truncateName(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s + strings.Repeat(" ", maxLen-len(s))
	}
	return s[:maxLen-1] + "…"
}

func renderGradientBar(brightness int, on bool, width int) string {
	if !on || brightness == 0 {
		return lipgloss.NewStyle().Foreground(styles.ColorLightOff).Render(strings.Repeat("─", width))
	}

	filled := (brightness * width) / 100
	if brightness > 0 && filled == 0 {
		filled = 1
	}

	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			// Gradient from dim orange to bright yellow
			intensity := 100 + (i * 155 / width)
			color := lipgloss.Color(fmt.Sprintf("#%02X%02X00", intensity, intensity/2))
			bar.WriteString(lipgloss.NewStyle().Foreground(color).Render("█"))
		} else {
			bar.WriteString(lipgloss.NewStyle().Foreground(styles.ColorLightOff).Render("─"))
		}
	}
	return bar.String()
}

func (m *Model) renderPanel(width int) string {
	// Show loading state
	if m.loading {
		return styles.StylePanel.Width(width - 4).Render(m.spinner.View() + " Loading...")
	}

	item := m.selectedItem()
	if item == nil {
		return styles.StylePanel.Width(width - 4).Render(styles.StyleMuted.Render("No selection"))
	}

	barWidth := width - 10
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 20 {
		barWidth = 20
	}

	var b strings.Builder

	if item.isGroup {
		// Group panel
		group := item.group
		b.WriteString(styles.StyleSelected.Render(group.Name))
		b.WriteString("\n\n")

		// Status
		lightsOn := 0
		totalBrightness := 0
		for _, light := range group.Lights {
			if light.On {
				lightsOn++
				totalBrightness += light.BrightnessPct()
			}
		}

		if lightsOn == 0 {
			b.WriteString(styles.StyleLightOff.Render("○ All Off"))
		} else if lightsOn == len(group.Lights) {
			b.WriteString(styles.StyleLightOn.Render("● All On"))
		} else {
			b.WriteString(styles.StyleLightOn.Render(fmt.Sprintf("● %d/%d On", lightsOn, len(group.Lights))))
		}
		b.WriteString("\n\n")

		// Average brightness
		if lightsOn > 0 {
			avgBrightness := totalBrightness / lightsOn
			b.WriteString(styles.StyleMuted.Render("Avg Brightness: "))
			b.WriteString(fmt.Sprintf("%d%%\n", avgBrightness))
			b.WriteString(renderGradientBar(avgBrightness, true, barWidth))
		} else {
			b.WriteString(styles.StyleMuted.Render("Avg Brightness: --\n"))
			b.WriteString(renderGradientBar(0, false, barWidth))
		}
		b.WriteString("\n\n")

		// Lights list
		b.WriteString(styles.StyleMuted.Render("Lights:") + "\n")
		maxLights := 6
		for i, light := range group.Lights {
			if i >= maxLights {
				b.WriteString(fmt.Sprintf("  ... +%d more\n", len(group.Lights)-maxLights))
				break
			}
			icon := styles.StyleLightOff.Render("○")
			if light.On {
				icon = styles.StyleLightOn.Render("●")
			}
			name := light.Name
			if len(name) > width-10 {
				name = name[:width-11] + "…"
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", icon, name))
		}

		b.WriteString("\n")
		b.WriteString(styles.StyleMuted.Render("←→ dim • space toggle"))

	} else {
		// Light panel
		light := item.light
		b.WriteString(styles.StyleSelected.Render(light.Name))
		b.WriteString("\n\n")

		// Status
		if light.On {
			b.WriteString(styles.StyleLightOn.Render("● On"))
		} else {
			b.WriteString(styles.StyleLightOff.Render("○ Off"))
		}
		b.WriteString("\n\n")

		// Brightness
		b.WriteString(styles.StyleMuted.Render("Brightness: "))
		b.WriteString(fmt.Sprintf("%d%%\n", light.BrightnessPct()))
		b.WriteString(renderGradientBar(light.BrightnessPct(), light.On, barWidth))
		b.WriteString("\n\n")

		// Color temperature
		b.WriteString(styles.StyleMuted.Render("Temperature: "))
		b.WriteString(fmt.Sprintf("%dK\n", light.Color.Kelvin))
		b.WriteString(renderTempBar(light.Color.Kelvin, barWidth))
		b.WriteString("\n")
		b.WriteString(styles.StyleMuted.Render("     warm ← → cool\n"))
		b.WriteString("\n")

		// Color preview
		hex := light.Color.Hex()
		colorBox := lipgloss.NewStyle().
			Background(lipgloss.Color(hex)).
			Render("    ")
		b.WriteString(styles.StyleMuted.Render("Color: "))
		b.WriteString(colorBox)
		b.WriteString("\n\n")

		// Presets
		b.WriteString(styles.StyleMuted.Render("Presets (p/P):") + "\n")
		for i, preset := range m.colorPresets {
			presetHex := preset.Color.Hex()
			colorDot := lipgloss.NewStyle().Foreground(lipgloss.Color(presetHex)).Render("●")
			var indicator, name string
			if i == m.presetIndex {
				indicator = styles.StyleSelected.Render("▸")
				name = styles.StyleSelected.Render(preset.Name)
			} else {
				indicator = " "
				name = preset.Name
			}
			b.WriteString(fmt.Sprintf("%s %s %s\n", colorDot, indicator, name))
		}
	}

	return styles.StylePanel.Width(width - 4).Render(b.String())
}

func renderTempBar(kelvin int, width int) string {
	// Kelvin range: 2500 (warm) to 9000 (cool)
	// Map to bar position
	minK, maxK := 2500, 9000
	if kelvin < minK {
		kelvin = minK
	}
	if kelvin > maxK {
		kelvin = maxK
	}
	pos := (kelvin - minK) * width / (maxK - minK)
	if pos >= width {
		pos = width - 1
	}

	var bar strings.Builder
	for i := 0; i < width; i++ {
		// Gradient from warm orange to cool blue
		ratio := float64(i) / float64(width)
		r := uint8(255 * (1 - ratio))
		g := uint8(180 - 80*ratio)
		b := uint8(100 + 155*ratio)

		char := "─"
		if i == pos {
			char = "●"
		}
		bar.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))).Render(char))
	}
	return bar.String()
}

func (m *Model) renderStatusBar() string {
	// Count lights on and active groups
	lightsOn := 0
	totalLights := 0
	activeGroups := make(map[string]bool)

	for _, item := range m.items {
		if !item.isGroup && item.light != nil {
			totalLights++
			if item.light.On {
				lightsOn++
				if item.light.GroupID != "" {
					activeGroups[item.light.GroupID] = true
				}
			}
		}
	}

	// Build status
	status := fmt.Sprintf("%d/%d lights on", lightsOn, totalLights)
	if len(m.groups) > 0 {
		status += fmt.Sprintf(" • %d/%d groups active", len(activeGroups), len(m.groups))
	}

	if m.status.text != "" {
		switch m.status.level {
		case statusWarn:
			return styles.StyleStatusWarn.Render(m.status.text)
		case statusErr:
			return styles.StyleStatusErr.Render(m.status.text)
		default:
			return styles.StyleStatusOk.Render(m.status.text)
		}
	}

	return styles.StyleMuted.Render(status)
}

func (m *Model) renderHelp() string {
	keys := []string{
		styles.StyleHelpKey.Render("↑↓") + " nav",
		styles.StyleHelpKey.Render("←→") + " dim",
		styles.StyleHelpKey.Render("space") + " toggle",
		styles.StyleHelpKey.Render("w/c") + " temp",
		styles.StyleHelpKey.Render("p") + " color",
		styles.StyleHelpKey.Render("a/x") + " group",
		styles.StyleHelpKey.Render("s") + " scenes",
		styles.StyleHelpKey.Render("/") + " search",
		styles.StyleHelpKey.Render("q") + " quit",
	}

	// Responsive help - show fewer items on narrow terminals
	if m.width < 60 {
		keys = []string{
			styles.StyleHelpKey.Render("↑↓") + " nav",
			styles.StyleHelpKey.Render("space") + " toggle",
			styles.StyleHelpKey.Render("q") + " quit",
		}
	} else if m.width < 90 {
		keys = []string{
			styles.StyleHelpKey.Render("↑↓") + " nav",
			styles.StyleHelpKey.Render("←→") + " dim",
			styles.StyleHelpKey.Render("space") + " toggle",
			styles.StyleHelpKey.Render("s") + " scenes",
			styles.StyleHelpKey.Render("q") + " quit",
		}
	}

	return styles.StyleHelp.Render(strings.Join(keys, "  "))
}

func renderBar(value, width int) string {
	if width <= 0 {
		return ""
	}
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	filled := int(float64(width) * float64(value) / 100.0)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return styles.StyleMuted.Render(strings.Repeat("█", filled)) + styles.StyleDim.Render(strings.Repeat("░", width-filled))
}

func brightnessFromKey(key string) int {
	switch key {
	case "0":
		return 100
	case "1":
		return 10
	case "2":
		return 20
	case "3":
		return 30
	case "4":
		return 40
	case "5":
		return 50
	case "6":
		return 60
	case "7":
		return 70
	case "8":
		return 80
	case "9":
		return 90
	default:
		return 100
	}
}

func loadPresets() []colorPreset {
	// Try to load from config file
	cfg, err := config.Load()
	if err == nil && cfg != nil && len(cfg.Presets) > 0 {
		presets := make([]colorPreset, len(cfg.Presets))
		for i, p := range cfg.Presets {
			presets[i] = colorPreset{
				Name: p.Name,
				Color: models.Color{
					Hue:        p.Hue,
					Saturation: p.Saturation,
					Kelvin:     p.Kelvin,
				},
			}
		}
		return presets
	}

	// Fall back to defaults
	return []colorPreset{
		{Name: "Sunrise", Color: models.Color{Hue: 35, Saturation: 80, Kelvin: 2500}}, // Warm orange glow
		{Name: "Morning", Color: models.Color{Hue: 40, Saturation: 30, Kelvin: 4000}}, // Soft warm white
		{Name: "Daylight", Color: models.Color{Hue: 0, Saturation: 0, Kelvin: 5500}},  // Bright neutral white
		{Name: "Sunset", Color: models.Color{Hue: 18, Saturation: 100, Kelvin: 2700}}, // Deep orange/red
		{Name: "Evening", Color: models.Color{Hue: 30, Saturation: 60, Kelvin: 2700}}, // Warm amber
		{Name: "Night", Color: models.Color{Hue: 15, Saturation: 100, Kelvin: 2000}},  // Dim warm red
	}
}
