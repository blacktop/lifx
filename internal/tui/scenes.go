package tui

import (
	"strings"

	"github.com/blacktop/lifx/internal/models"
	"github.com/blacktop/lifx/internal/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sceneItem struct {
	Scene   *models.Scene
	Header  string
	IsTitle bool
}

type sceneActivateMsg struct {
	sceneID string
}

type closeScenesMsg struct{}

// ScenesModel renders the scenes modal.
type ScenesModel struct {
	scenes    []*models.Scene
	locations []*models.Location
	items     []sceneItem
	selected  int
	width     int
	height    int
}

// NewScenesModel creates a new scenes model.
func NewScenesModel() ScenesModel {
	return ScenesModel{}
}

// SetScenes updates the scenes list.
func (m *ScenesModel) SetScenes(scenes []*models.Scene, locations []*models.Location) {
	m.scenes = scenes
	m.locations = locations
	m.rebuild()
}

func (m *ScenesModel) rebuild() {
	m.items = nil
	locationNames := map[string]string{}
	for _, loc := range m.locations {
		locationNames[loc.ID] = loc.Name
	}
	grouped := models.ScenesByLocation(m.scenes)
	for locationID, sceneList := range grouped {
		header := locationNames[locationID]
		if header == "" {
			header = "Scenes"
		}
		m.items = append(m.items, sceneItem{Header: header, IsTitle: true})
		for _, scene := range sceneList {
			m.items = append(m.items, sceneItem{Scene: scene})
		}
	}
	m.selected = 0
	for i, item := range m.items {
		if !item.IsTitle {
			m.selected = i
			break
		}
	}
}

// Update handles key events.
func (m ScenesModel) Update(msg tea.Msg) (ScenesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "s":
			return m, func() tea.Msg { return closeScenesMsg{} }
		case "up", "k":
			m.movePrev()
		case "down", "j":
			m.moveNext()
		case "enter":
			if m.selected >= 0 && m.selected < len(m.items) {
				item := m.items[m.selected]
				if item.Scene != nil {
					return m, func() tea.Msg { return sceneActivateMsg{sceneID: item.Scene.ID} }
				}
			}
		}
	}
	return m, nil
}

func (m *ScenesModel) moveNext() {
	for i := m.selected + 1; i < len(m.items); i++ {
		if !m.items[i].IsTitle {
			m.selected = i
			return
		}
	}
}

func (m *ScenesModel) movePrev() {
	for i := m.selected - 1; i >= 0; i-- {
		if !m.items[i].IsTitle {
			m.selected = i
			return
		}
	}
}

// View renders the modal.
func (m ScenesModel) View() string {
	var b strings.Builder
	b.WriteString(styles.StylePanelTitle.Render("Scenes"))
	b.WriteString("\n\n")

	if len(m.items) == 0 {
		b.WriteString(styles.StyleMuted.Render("No scenes available"))
		return modalStyle().Render(b.String())
	}

	for i, item := range m.items {
		if item.IsTitle {
			b.WriteString(styles.StyleGroupTitle.Render(item.Header))
			b.WriteString("\n")
			continue
		}
		cursor := "  "
		style := styles.StyleMuted
		if i == m.selected {
			cursor = "▸ "
			style = styles.StyleSelected
		}
		b.WriteString(cursor + style.Render(item.Scene.Name))
		b.WriteString("\n")
	}

	return modalStyle().Render(b.String())
}

func modalStyle() lipgloss.Style {
	return styles.StylePanel.Copy().
		Border(lipgloss.DoubleBorder()).
		Width(50)
}
