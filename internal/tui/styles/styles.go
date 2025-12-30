package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color palette - Lavender theme inspired by hue-tui
var (
	// Primary colors
	ColorPrimary    = lipgloss.Color("#B794F4") // Lavender
	ColorSecondary  = lipgloss.Color("#9F7AEA") // Darker lavender
	ColorAccent     = lipgloss.Color("#E9D8FD") // Light lavender
	ColorBackground = lipgloss.Color("#1A1A2E") // Dark background
	ColorSurface    = lipgloss.Color("#2D2D44") // Surface color
	ColorSurfaceAlt = lipgloss.Color("#3D3D5C") // Alternate surface

	// Text colors
	ColorText        = lipgloss.Color("#FAFAFA") // Primary text
	ColorTextMuted   = lipgloss.Color("#A0A0B0") // Muted text
	ColorTextDim     = lipgloss.Color("#6B6B80") // Dim text
	ColorTextInverse = lipgloss.Color("#1A1A2E") // Inverse text

	// State colors
	ColorSuccess = lipgloss.Color("#68D391") // Green
	ColorWarning = lipgloss.Color("#F6E05E") // Yellow
	ColorError   = lipgloss.Color("#FC8181") // Red
	ColorInfo    = lipgloss.Color("#63B3ED") // Blue

	// Light states
	ColorLightOn  = lipgloss.Color("#FBBF24") // Warm yellow for on
	ColorLightOff = lipgloss.Color("#4A4A5A") // Gray for off
)

// Styles for various UI components
var (
	// Base styles
	StyleBase = lipgloss.NewStyle().
			Background(ColorBackground).
			Foreground(ColorText)

	// Header styles
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorPrimary).
			Padding(0, 1)

	StyleHeaderBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorTextInverse).
				Background(ColorSecondary).
				Padding(0, 1).
				MarginLeft(1)

	// Group/Room styles
	StyleGroupTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// Light styles
	StyleLightOn = lipgloss.NewStyle().
			Foreground(ColorLightOn)

	StyleLightOff = lipgloss.NewStyle().
			Foreground(ColorLightOff)

	StyleLightName = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleLightNameDim = lipgloss.NewStyle().
				Foreground(ColorTextMuted)

	// Selection
	StyleSelected = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Panel styles
	StylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	StylePanelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			MarginBottom(1)

	// Status styles
	StyleStatusOk = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleStatusWarn = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleStatusErr = lipgloss.NewStyle().
			Foreground(ColorError)

	// Help styles
	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	StyleHelpKey = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Search styles
	StyleSearch = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Muted/Dim text
	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	// Modal styles
	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Background(ColorSurface).
			Padding(1, 2)

	StyleModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Scene styles
	StyleSceneItem = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	StyleSceneItemSelected = lipgloss.NewStyle().
				Foreground(ColorTextInverse).
				Background(ColorPrimary).
				Padding(0, 1)

	// Spinner style
	StyleSpinner = lipgloss.NewStyle().
			Foreground(ColorPrimary)
)

// GetBrightnessColor returns a color for the brightness gradient
func GetBrightnessColor(position int, total int) lipgloss.Color {
	// Gradient from dim orange to bright yellow
	intensity := 100 + (position * 155 / total)
	return lipgloss.Color(fmt.Sprintf("#%02X%02X00", intensity, intensity/2))
}
