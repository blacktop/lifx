package tui

import (
	"context"
	"time"

	"github.com/blacktop/lifx/internal/backend/selector"
	tea "github.com/charmbracelet/bubbletea"
)

// Options configures the TUI.
type Options struct {
	Backend      string
	APIKey       string
	LanListen    string
	LanBroadcast string
	AutoRefresh  time.Duration
	Debug        bool
}

// Run launches the TUI.
func Run(ctx context.Context, opts Options) error {
	be, warn, err := selector.Select(ctx, selector.Options{
		Backend:      opts.Backend,
		APIKey:       opts.APIKey,
		LanListen:    opts.LanListen,
		LanBroadcast: opts.LanBroadcast,
		Debug:        opts.Debug,
	})
	if err != nil {
		return err
	}

	model := NewModel(ctx, be, warn, opts.AutoRefresh)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}
