package models

// Group represents a LIFX group (room/zone).
type Group struct {
	ID           string
	Name         string
	LocationID   string
	LocationName string
	Lights       []*Light
	AllOn        bool
	AnyOn        bool
}

// UpdateState recalculates AllOn and AnyOn based on light states.
func (g *Group) UpdateState() {
	if len(g.Lights) == 0 {
		g.AllOn = false
		g.AnyOn = false
		return
	}
	g.AllOn = true
	g.AnyOn = false
	for _, light := range g.Lights {
		if light.On {
			g.AnyOn = true
		} else {
			g.AllOn = false
		}
	}
}

// LightByID finds a light in this group by ID.
func (g *Group) LightByID(id string) *Light {
	for _, light := range g.Lights {
		if light.ID == id {
			return light
		}
	}
	return nil
}

// AverageBrightness returns the average brightness of lights that are on.
func (g *Group) AverageBrightness() int {
	if len(g.Lights) == 0 {
		return 0
	}
	var total int
	var count int
	for _, light := range g.Lights {
		if light.On {
			total += light.BrightnessPct()
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / count
}
