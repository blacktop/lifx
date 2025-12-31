package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/blacktop/lifx/internal/backend"
	"github.com/blacktop/lifx/internal/models"
)

const baseURL = "https://api.lifx.com/v1"

// Backend implements the LIFX HTTP API backend.
type Backend struct {
	apiKey string
	client *http.Client
}

// New creates a new API backend.
func New(apiKey string) *Backend {
	return &Backend{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (b *Backend) Type() backend.Type {
	return backend.BackendAPI
}

func (b *Backend) Name() string {
	return "Cloud API"
}

func (b *Backend) SupportsScenes() bool {
	return true
}

func (b *Backend) Init(ctx context.Context) error {
	if b.apiKey == "" {
		return errors.New("missing LIFX API key (set LIFX_API_KEY or use --api-key)")
	}
	// Validate the API key by listing lights
	_, err := b.listLights(ctx, "all")
	if err != nil {
		return fmt.Errorf("failed to validate API key: %w", err)
	}
	return nil
}

func (b *Backend) Close() error {
	return nil
}

func (b *Backend) Refresh(ctx context.Context) (*backend.State, error) {
	lights, err := b.listLights(ctx, "all")
	if err != nil {
		return nil, err
	}

	scenes, err := b.listScenes(ctx)
	if err != nil {
		// Scenes are optional, don't fail
		scenes = nil
	}

	// Group lights by group and location
	locationMap := make(map[string]*models.Location)
	groupMap := make(map[string]*models.Group)

	for _, light := range lights {
		// Build location
		if light.Location.ID != "" {
			if _, ok := locationMap[light.Location.ID]; !ok {
				locationMap[light.Location.ID] = &models.Location{
					ID:   light.Location.ID,
					Name: light.Location.Name,
				}
			}
		}

		// Build group
		groupID := light.Group.ID
		if groupID == "" {
			groupID = "ungrouped"
		}
		group, ok := groupMap[groupID]
		if !ok {
			group = &models.Group{
				ID:           groupID,
				Name:         light.Group.Name,
				LocationID:   light.Location.ID,
				LocationName: light.Location.Name,
			}
			if group.Name == "" {
				group.Name = "Ungrouped"
			}
			groupMap[groupID] = group
		}

		// Convert to model
		model := &models.Light{
			ID:         light.ID,
			Name:       light.Label,
			On:         light.Power == "on",
			Brightness: int(light.Brightness * 100),
			Color: models.Color{
				Hue:        int(light.Color.Hue),
				Saturation: int(light.Color.Saturation * 100),
				Brightness: int(light.Brightness * 100),
				Kelvin:     light.Color.Kelvin,
			},
			GroupID:    groupID,
			LocationID: light.Location.ID,
			Reachable:  light.Connected,
		}
		group.Lights = append(group.Lights, model)
	}

	// Convert maps to slices
	locations := make([]*models.Location, 0, len(locationMap))
	for _, loc := range locationMap {
		locations = append(locations, loc)
	}

	groups := make([]*models.Group, 0, len(groupMap))
	for _, group := range groupMap {
		group.UpdateState()
		groups = append(groups, group)
	}

	// Convert scenes
	sceneModels := make([]*models.Scene, 0, len(scenes))
	for _, scene := range scenes {
		sceneModels = append(sceneModels, &models.Scene{
			ID:   scene.UUID,
			Name: scene.Name,
		})
	}

	return &backend.State{
		Locations: locations,
		Groups:    groups,
		Scenes:    sceneModels,
	}, nil
}

func (b *Backend) SetLightPower(ctx context.Context, lightID string, on bool) error {
	power := "off"
	if on {
		power = "on"
	}
	return b.setState(ctx, "id:"+lightID, map[string]any{
		"power":    power,
		"duration": 0.2,
	})
}

func (b *Backend) SetAllPower(ctx context.Context, on bool) error {
	power := "off"
	if on {
		power = "on"
	}
	return b.setState(ctx, "all", map[string]any{
		"power":    power,
		"duration": 0.2,
	})
}

func (b *Backend) SetLightColor(ctx context.Context, lightID string, color models.Color) error {
	c := color.Clamp()
	return b.setState(ctx, "id:"+lightID, map[string]any{
		"color":      fmt.Sprintf("hue:%d saturation:%f", c.Hue, float64(c.Saturation)/100.0),
		"brightness": float64(c.Brightness) / 100.0,
		"kelvin":     c.Kelvin,
		"duration":   0.2,
	})
}

func (b *Backend) ActivateScene(ctx context.Context, sceneID string) error {
	url := fmt.Sprintf("%s/scenes/scene_id:%s/activate", baseURL, sceneID)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return err
	}
	b.setHeaders(req)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// API types

type apiLight struct {
	ID         string   `json:"id"`
	UUID       string   `json:"uuid"`
	Label      string   `json:"label"`
	Connected  bool     `json:"connected"`
	Power      string   `json:"power"`
	Color      apiColor `json:"color"`
	Brightness float64  `json:"brightness"`
	Group      struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"group"`
	Location struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"location"`
	Product struct {
		Name         string `json:"name"`
		Identifier   string `json:"identifier"`
		Company      string `json:"company"`
		Capabilities struct {
			HasColor             bool `json:"has_color"`
			HasVariableColorTemp bool `json:"has_variable_color_temp"`
			HasIR                bool `json:"has_ir"`
			HasChain             bool `json:"has_chain"`
			HasMatrix            bool `json:"has_matrix"`
			HasMultizone         bool `json:"has_multizone"`
			MinKelvin            int  `json:"min_kelvin"`
			MaxKelvin            int  `json:"max_kelvin"`
		} `json:"capabilities"`
	} `json:"product"`
}

type apiColor struct {
	Hue        float64 `json:"hue"`
	Saturation float64 `json:"saturation"`
	Kelvin     int     `json:"kelvin"`
}

type apiScene struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Account struct {
		UUID string `json:"uuid"`
	} `json:"account"`
	States []struct {
		Selector   string   `json:"selector"`
		Power      string   `json:"power,omitempty"`
		Brightness float64  `json:"brightness,omitempty"`
		Color      apiColor `json:"color,omitempty"`
	} `json:"states"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

func (b *Backend) listLights(ctx context.Context, selector string) ([]apiLight, error) {
	url := fmt.Sprintf("%s/lights/%s", baseURL, selector)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	b.setHeaders(req)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, errors.New("invalid API key")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var lights []apiLight
	if err := json.NewDecoder(resp.Body).Decode(&lights); err != nil {
		return nil, err
	}
	return lights, nil
}

func (b *Backend) listScenes(ctx context.Context) ([]apiScene, error) {
	url := fmt.Sprintf("%s/scenes", baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	b.setHeaders(req)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var scenes []apiScene
	if err := json.NewDecoder(resp.Body).Decode(&scenes); err != nil {
		return nil, err
	}
	return scenes, nil
}

func (b *Backend) setState(ctx context.Context, selector string, state map[string]any) error {
	url := fmt.Sprintf("%s/lights/%s/state", baseURL, selector)

	body, err := json.Marshal(state)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	b.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (b *Backend) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("Accept", "application/json")
}
