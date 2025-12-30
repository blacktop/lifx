package models

// Scene represents a LIFX scene.
type Scene struct {
	ID         string
	Name       string
	LocationID string
	Location   string
}

// ScenesByLocation groups scenes by location ID.
func ScenesByLocation(scenes []*Scene) map[string][]*Scene {
	grouped := make(map[string][]*Scene)
	for _, scene := range scenes {
		grouped[scene.LocationID] = append(grouped[scene.LocationID], scene)
	}
	return grouped
}
