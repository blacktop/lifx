package models

// Light represents a LIFX light.
type Light struct {
	ID         string
	Name       string
	On         bool
	Brightness int
	Color      Color
	GroupID    string
	LocationID string
	Reachable  bool
}

// BrightnessPct returns the brightness percentage (0-100).
func (l *Light) BrightnessPct() int {
	if l.Brightness < 0 {
		return 0
	}
	if l.Brightness > 100 {
		return 100
	}
	return l.Brightness
}

// SetBrightnessPct updates brightness and keeps color in sync.
func (l *Light) SetBrightnessPct(pct int) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	l.Brightness = pct
	l.Color.Brightness = pct
}

// Clone creates a shallow copy of the light.
func (l *Light) Clone() *Light {
	clone := *l
	return &clone
}
