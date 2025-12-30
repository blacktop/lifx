package backend

import (
	"context"

	"github.com/blacktop/lifx/internal/models"
)

// Type identifies a backend implementation.
type Type string

const (
	BackendAuto Type = "auto"
	BackendLAN  Type = "lan"
	BackendAPI  Type = "api"
)

// State is a snapshot of the backend state.
type State struct {
	Locations []*models.Location
	Groups    []*models.Group
	Scenes    []*models.Scene
}

// Backend defines the operations required by the TUI.
type Backend interface {
	Type() Type
	Name() string
	SupportsScenes() bool
	Init(ctx context.Context) error
	Close() error
	Refresh(ctx context.Context) (*State, error)
	SetLightPower(ctx context.Context, lightID string, on bool) error
	SetLightColor(ctx context.Context, lightID string, color models.Color) error
	ActivateScene(ctx context.Context, sceneID string) error
}
