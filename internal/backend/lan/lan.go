package lan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"github.com/blacktop/lifx/internal/backend"
	"github.com/blacktop/lifx/internal/models"
	"github.com/pdf/golifx"
	"github.com/pdf/golifx/common"
	"github.com/pdf/golifx/protocol"
)

// Options controls LAN backend behavior.
type Options struct {
	Debug       bool
	ListenIP    string
	BroadcastIP string
}

// Backend implements the LIFX LAN backend.
type Backend struct {
	client      *golifx.Client
	lights      map[string]common.Light
	lightsMu    sync.RWMutex
	debug       bool
	logger      *log.Logger
	startedAt   time.Time
	listenIP    string
	broadcastIP string
}

// New creates a LAN backend.
func New(opts Options) *Backend {
	return &Backend{
		debug:       opts.Debug,
		listenIP:    opts.ListenIP,
		broadcastIP: opts.BroadcastIP,
	}
}

func (b *Backend) Type() backend.Type {
	return backend.BackendLAN
}

func (b *Backend) Name() string {
	return "LAN"
}

func (b *Backend) SupportsScenes() bool {
	return false
}

func (b *Backend) Init(ctx context.Context) error {
	if b.debug {
		b.logger = log.New(os.Stderr, "lifx-lan ", log.LstdFlags|log.Lmicroseconds)
		golifx.SetLogger(&stdLogger{log: b.logger})
	}

	proto := &protocol.V2{Reliable: true}
	if b.listenIP != "" {
		ip := net.ParseIP(b.listenIP)
		if ip == nil {
			return fmt.Errorf("invalid listen IP: %s", b.listenIP)
		}
		proto.IP = ip
	}
	// Note: broadcastIP is not supported by upstream golifx; ignored for now

	client, err := golifx.NewClient(proto)
	if err != nil {
		return err
	}
	if err := client.SetDiscoveryInterval(5 * time.Second); err != nil {
		return err
	}
	b.client = client
	b.startedAt = time.Now()
	return nil
}

func (b *Backend) Close() error {
	if b.client == nil {
		return nil
	}
	return b.client.Close()
}

func (b *Backend) Refresh(ctx context.Context) (*backend.State, error) {
	if b.client == nil {
		return nil, errors.New("lan backend not initialized")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	locations, _ := b.client.GetLocations()
	groups, _ := b.client.GetGroups()
	allLights, _ := b.client.GetLights()

	locationModels := make([]*models.Location, 0, len(locations))
	locationByLight := make(map[string]*models.Location)

	for _, location := range locations {
		locModel := &models.Location{ID: location.ID(), Name: location.GetLabel()}
		locationModels = append(locationModels, locModel)
		for _, light := range location.Lights() {
			locationByLight[lightID(light)] = locModel
		}
	}

	lightModels := make(map[string]*models.Light)
	groupModels := make([]*models.Group, 0, len(groups))

	for _, group := range groups {
		groupModel := &models.Group{ID: group.ID(), Name: group.GetLabel()}
		for _, light := range group.Lights() {
			model := lightModels[lightID(light)]
			if model == nil {
				model = b.lightToModel(light)
				lightModels[model.ID] = model
			}
			model.GroupID = groupModel.ID
			if loc := locationByLight[model.ID]; loc != nil {
				model.LocationID = loc.ID
				groupModel.LocationID = loc.ID
				groupModel.LocationName = loc.Name
			}
			groupModel.Lights = append(groupModel.Lights, model)
		}
		groupModel.UpdateState()
		groupModels = append(groupModels, groupModel)
	}

	// Add lights not in any group to an Ungrouped bucket.
	ungrouped := &models.Group{ID: "ungrouped", Name: "Ungrouped"}
	for _, light := range allLights {
		id := lightID(light)
		model := lightModels[id]
		if model == nil {
			model = b.lightToModel(light)
			lightModels[model.ID] = model
		}
		if model.GroupID == "" {
			if loc := locationByLight[model.ID]; loc != nil {
				model.LocationID = loc.ID
				ungrouped.LocationID = loc.ID
				ungrouped.LocationName = loc.Name
			}
			ungrouped.Lights = append(ungrouped.Lights, model)
		}
	}
	if len(ungrouped.Lights) > 0 {
		ungrouped.UpdateState()
		groupModels = append(groupModels, ungrouped)
	}

	b.lightsMu.Lock()
	b.lights = make(map[string]common.Light)
	for _, light := range allLights {
		b.lights[lightID(light)] = light
	}
	b.lightsMu.Unlock()

	return &backend.State{
		Locations: locationModels,
		Groups:    groupModels,
		Scenes:    nil,
	}, nil
}

func (b *Backend) SetLightPower(ctx context.Context, lightID string, on bool) error {
	light, err := b.lookupLight(lightID)
	if err != nil {
		return err
	}
	return light.SetPowerDuration(on, 200*time.Millisecond)
}

func (b *Backend) SetLightColor(ctx context.Context, lightID string, color models.Color) error {
	light, err := b.lookupLight(lightID)
	if err != nil {
		return err
	}
	return light.SetColor(colorToCommon(color), 200*time.Millisecond)
}

func (b *Backend) ActivateScene(ctx context.Context, sceneID string) error {
	return errors.New("scenes are not available over LAN")
}

func (b *Backend) lookupLight(id string) (common.Light, error) {
	b.lightsMu.RLock()
	light := b.lights[id]
	b.lightsMu.RUnlock()
	if light == nil {
		return nil, fmt.Errorf("unknown light: %s", id)
	}
	return light, nil
}

func (b *Backend) lightToModel(light common.Light) *models.Light {
	label, err := light.GetLabel()
	if err != nil || label == "" {
		label = fmt.Sprintf("Light %d", light.ID())
	}

	power, err := light.GetPower()
	if err != nil {
		power = light.CachedPower()
	}

	color, err := light.GetColor()
	if err != nil {
		color = light.CachedColor()
	}

	model := &models.Light{
		ID:         lightID(light),
		Name:       label,
		On:         power,
		Brightness: hsbBrightness(color),
		Color:      colorFromCommon(color),
		Reachable:  err == nil,
	}
	model.Color.Brightness = model.Brightness
	return model
}

func lightID(light common.Light) string {
	return fmt.Sprintf("%d", light.ID())
}

func hsbBrightness(color common.Color) int {
	b := float64(color.Brightness) / float64(math.MaxUint16) * 100.0
	return int(math.Round(b))
}

func colorFromCommon(color common.Color) models.Color {
	h := float64(color.Hue) / float64(math.MaxUint16) * 360.0
	s := float64(color.Saturation) / float64(math.MaxUint16) * 100.0
	b := float64(color.Brightness) / float64(math.MaxUint16) * 100.0
	return models.Color{
		Hue:        int(math.Round(h)),
		Saturation: int(math.Round(s)),
		Brightness: int(math.Round(b)),
		Kelvin:     int(color.Kelvin),
	}.Clamp()
}

func colorToCommon(color models.Color) common.Color {
	c := color.Clamp()
	h := uint16(float64(c.Hue) / 360.0 * float64(math.MaxUint16))
	s := uint16(float64(c.Saturation) / 100.0 * float64(math.MaxUint16))
	b := uint16(float64(c.Brightness) / 100.0 * float64(math.MaxUint16))
	return common.Color{
		Hue:        h,
		Saturation: s,
		Brightness: b,
		Kelvin:     uint16(c.Kelvin),
	}
}

// stdLogger adapts the standard log.Logger to golifx's logger interface.
type stdLogger struct {
	log *log.Logger
}

func (l *stdLogger) Debugf(format string, args ...any) { l.log.Printf(format, args...) }
func (l *stdLogger) Infof(format string, args ...any)  { l.log.Printf(format, args...) }
func (l *stdLogger) Warnf(format string, args ...any)  { l.log.Printf(format, args...) }
func (l *stdLogger) Errorf(format string, args ...any) { l.log.Printf(format, args...) }
func (l *stdLogger) Fatalf(format string, args ...any) {
	l.log.Printf(format, args...)
	os.Exit(1)
}
func (l *stdLogger) Panicf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }
