package selector

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/blacktop/lifx/internal/backend"
	"github.com/blacktop/lifx/internal/backend/api"
	"github.com/blacktop/lifx/internal/backend/lan"
)

// Options control backend selection.
type Options struct {
	Backend      string
	APIKey       string
	Debug        bool
	LanListen    string
	LanBroadcast string
}

// Select returns an initialized backend and an optional warning.
func Select(ctx context.Context, opts Options) (backend.Backend, string, error) {
	backendChoice := strings.ToLower(strings.TrimSpace(opts.Backend))
	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("LIFX_API_KEY"))
	}

	switch backendChoice {
	case "", string(backend.BackendAuto):
		if apiKey != "" {
			apiBackend := api.New(apiKey)
			if err := apiBackend.Init(ctx); err == nil {
				return apiBackend, "", nil
			}
			warn := "API backend unavailable; falling back to LAN"
			lanBackend := lan.New(lan.Options{Debug: opts.Debug, ListenIP: opts.LanListen, BroadcastIP: opts.LanBroadcast})
			if err := lanBackend.Init(ctx); err != nil {
				return nil, warn, err
			}
			return lanBackend, warn, nil
		}

		lanBackend := lan.New(lan.Options{Debug: opts.Debug, ListenIP: opts.LanListen, BroadcastIP: opts.LanBroadcast})
		if err := lanBackend.Init(ctx); err != nil {
			return nil, "", err
		}
		return lanBackend, "", nil

	case string(backend.BackendLAN):
		lanBackend := lan.New(lan.Options{Debug: opts.Debug, ListenIP: opts.LanListen, BroadcastIP: opts.LanBroadcast})
		if err := lanBackend.Init(ctx); err != nil {
			return nil, "", err
		}
		return lanBackend, "", nil

	case string(backend.BackendAPI):
		apiBackend := api.New(apiKey)
		if err := apiBackend.Init(ctx); err != nil {
			return nil, "", err
		}
		return apiBackend, "", nil

	default:
		return nil, "", fmt.Errorf("unknown backend: %s", backendChoice)
	}
}
