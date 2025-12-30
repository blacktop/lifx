/*
Copyright © 2025 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/blacktop/lifx/internal/backend/api"
	"github.com/blacktop/lifx/internal/models"
	"github.com/caarlos0/ctrlc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// Version can be set at build time
var Version = "dev"

func init() {
	rootCmd.AddCommand(mcpCmd)
}

var mcpCmd = &cobra.Command{
	Use:    "mcp",
	Short:  "Run as an MCP server",
	Long:   `Run lifx as an MCP (Model Context Protocol) server for AI agent integration.`,
	Hidden: true,
	RunE:   runMCP,
}

// MCP tool parameter types
type ListLightsParams struct{}

type SetPowerParams struct {
	LightID string `json:"light_id"`
	Power   bool   `json:"power"`
}

type SetBrightnessParams struct {
	LightID    string `json:"light_id"`
	Brightness int    `json:"brightness"`
}

type SetColorParams struct {
	LightID    string `json:"light_id"`
	Hue        *int   `json:"hue,omitempty"`
	Saturation *int   `json:"saturation,omitempty"`
	Brightness *int   `json:"brightness,omitempty"`
	Kelvin     *int   `json:"kelvin,omitempty"`
}

type ListScenesParams struct{}

type ActivateSceneParams struct {
	SceneID string `json:"scene_id"`
}

type SetAllPowerParams struct {
	Power bool `json:"power"`
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Get API key
	apiKey := flagAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("LIFX_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("LIFX_API_KEY environment variable or --api-key flag required for MCP mode")
	}

	// Create API backend
	backend := api.New(apiKey)

	// Initialize backend
	ctx := cmd.Context()
	if err := backend.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize LIFX API: %w", err)
	}
	defer backend.Close()

	// Create MCP server
	impl := &mcp.Implementation{
		Name:    "lifx-mcp",
		Title:   "LIFX Light Control",
		Version: Version,
	}
	s := mcp.NewServer(impl, nil)

	// Add list_lights tool
	listLightsTool := &mcp.Tool{
		Name:        "list_lights",
		Title:       "List Lights",
		Description: "List all LIFX lights with their current state (power, brightness, color)",
		InputSchema: buildListLightsSchema(),
	}
	mcp.AddTool(s, listLightsTool, func(ctx context.Context, req *mcp.CallToolRequest, input ListLightsParams) (*mcp.CallToolResult, any, error) {
		state, err := backend.Refresh(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		// Build response with all lights
		type LightInfo struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Power      string `json:"power"`
			Brightness int    `json:"brightness"`
			Hue        int    `json:"hue"`
			Saturation int    `json:"saturation"`
			Kelvin     int    `json:"kelvin"`
			Group      string `json:"group"`
			Reachable  bool   `json:"reachable"`
		}

		var lights []LightInfo
		for _, group := range state.Groups {
			for _, light := range group.Lights {
				power := "off"
				if light.On {
					power = "on"
				}
				lights = append(lights, LightInfo{
					ID:         light.ID,
					Name:       light.Name,
					Power:      power,
					Brightness: light.Brightness,
					Hue:        light.Color.Hue,
					Saturation: light.Color.Saturation,
					Kelvin:     light.Color.Kelvin,
					Group:      group.Name,
					Reachable:  light.Reachable,
				})
			}
		}

		jsonData, err := json.MarshalIndent(lights, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error marshaling response: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
		}, nil, nil
	})

	// Add set_power tool
	setPowerTool := &mcp.Tool{
		Name:        "set_power",
		Title:       "Set Power",
		Description: "Turn a LIFX light on or off",
		InputSchema: buildSetPowerSchema(),
	}
	mcp.AddTool(s, setPowerTool, func(ctx context.Context, req *mcp.CallToolRequest, input SetPowerParams) (*mcp.CallToolResult, any, error) {
		if input.LightID == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: light_id is required"}},
				IsError: true,
			}, nil, nil
		}

		if err := backend.SetLightPower(ctx, input.LightID, input.Power); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		state := "off"
		if input.Power {
			state = "on"
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Light %s turned %s", input.LightID, state)}},
		}, nil, nil
	})

	// Add set_all_power tool
	setAllPowerTool := &mcp.Tool{
		Name:        "set_all_power",
		Title:       "Set All Power",
		Description: "Turn all LIFX lights on or off",
		InputSchema: buildSetAllPowerSchema(),
	}
	mcp.AddTool(s, setAllPowerTool, func(ctx context.Context, req *mcp.CallToolRequest, input SetAllPowerParams) (*mcp.CallToolResult, any, error) {
		if err := backend.SetAllPower(ctx, input.Power); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		state := "off"
		if input.Power {
			state = "on"
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("All lights turned %s", state)}},
		}, nil, nil
	})

	// Add set_brightness tool
	setBrightnessTool := &mcp.Tool{
		Name:        "set_brightness",
		Title:       "Set Brightness",
		Description: "Set the brightness level of a LIFX light (0-100)",
		InputSchema: buildSetBrightnessSchema(),
	}
	mcp.AddTool(s, setBrightnessTool, func(ctx context.Context, req *mcp.CallToolRequest, input SetBrightnessParams) (*mcp.CallToolResult, any, error) {
		if input.LightID == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: light_id is required"}},
				IsError: true,
			}, nil, nil
		}

		// Clamp brightness
		brightness := input.Brightness
		if brightness < 0 {
			brightness = 0
		}
		if brightness > 100 {
			brightness = 100
		}

		// Get current state to preserve other color values
		state, err := backend.Refresh(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error getting light state: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		// Find the light
		var light *models.Light
		for _, group := range state.Groups {
			for _, l := range group.Lights {
				if l.ID == input.LightID {
					light = l
					break
				}
			}
		}
		if light == nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: light %s not found", input.LightID)}},
				IsError: true,
			}, nil, nil
		}

		// Update brightness
		color := light.Color
		color.Brightness = brightness
		if err := backend.SetLightColor(ctx, input.LightID, color); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Light %s brightness set to %d%%", input.LightID, brightness)}},
		}, nil, nil
	})

	// Add set_color tool
	setColorTool := &mcp.Tool{
		Name:        "set_color",
		Title:       "Set Color",
		Description: "Set the color of a LIFX light using hue, saturation, brightness, and/or kelvin temperature",
		InputSchema: buildSetColorSchema(),
	}
	mcp.AddTool(s, setColorTool, func(ctx context.Context, req *mcp.CallToolRequest, input SetColorParams) (*mcp.CallToolResult, any, error) {
		if input.LightID == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: light_id is required"}},
				IsError: true,
			}, nil, nil
		}

		// Get current state
		state, err := backend.Refresh(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error getting light state: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		// Find the light
		var light *models.Light
		for _, group := range state.Groups {
			for _, l := range group.Lights {
				if l.ID == input.LightID {
					light = l
					break
				}
			}
		}
		if light == nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: light %s not found", input.LightID)}},
				IsError: true,
			}, nil, nil
		}

		// Update color values
		color := light.Color
		if input.Hue != nil {
			color.Hue = *input.Hue
		}
		if input.Saturation != nil {
			color.Saturation = *input.Saturation
		}
		if input.Brightness != nil {
			color.Brightness = *input.Brightness
		}
		if input.Kelvin != nil {
			color.Kelvin = *input.Kelvin
		}

		// Clamp values
		color = color.Clamp()

		if err := backend.SetLightColor(ctx, input.LightID, color); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Light %s color updated (hue=%d, sat=%d, brightness=%d, kelvin=%d)",
				input.LightID, color.Hue, color.Saturation, color.Brightness, color.Kelvin)}},
		}, nil, nil
	})

	// Add list_scenes tool
	listScenesTool := &mcp.Tool{
		Name:        "list_scenes",
		Title:       "List Scenes",
		Description: "List all available LIFX scenes",
		InputSchema: buildListScenesSchema(),
	}
	mcp.AddTool(s, listScenesTool, func(ctx context.Context, req *mcp.CallToolRequest, input ListScenesParams) (*mcp.CallToolResult, any, error) {
		state, err := backend.Refresh(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		type SceneInfo struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		var scenes []SceneInfo
		for _, scene := range state.Scenes {
			scenes = append(scenes, SceneInfo{
				ID:   scene.ID,
				Name: scene.Name,
			})
		}

		if len(scenes) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "No scenes found"}},
			}, nil, nil
		}

		jsonData, err := json.MarshalIndent(scenes, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error marshaling response: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
		}, nil, nil
	})

	// Add activate_scene tool
	activateSceneTool := &mcp.Tool{
		Name:        "activate_scene",
		Title:       "Activate Scene",
		Description: "Activate a LIFX scene by its ID",
		InputSchema: buildActivateSceneSchema(),
	}
	mcp.AddTool(s, activateSceneTool, func(ctx context.Context, req *mcp.CallToolRequest, input ActivateSceneParams) (*mcp.CallToolResult, any, error) {
		if input.SceneID == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: scene_id is required"}},
				IsError: true,
			}, nil, nil
		}

		if err := backend.ActivateScene(ctx, input.SceneID); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Scene %s activated", input.SceneID)}},
		}, nil, nil
	})

	// Run server with graceful shutdown
	return ctrlc.Default.Run(ctx, func() error {
		return s.Run(ctx, &mcp.StdioTransport{})
	})
}

// Schema builders for MCP tools - return json.RawMessage

func buildListLightsSchema() json.RawMessage {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	data, _ := json.Marshal(schema)
	return data
}

func buildSetPowerSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"light_id": map[string]any{
				"type":        "string",
				"description": "The ID of the light to control",
			},
			"power": map[string]any{
				"type":        "boolean",
				"description": "true to turn on, false to turn off",
			},
		},
		"required": []string{"light_id", "power"},
	}
	data, _ := json.Marshal(schema)
	return data
}

func buildSetAllPowerSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"power": map[string]any{
				"type":        "boolean",
				"description": "true to turn all lights on, false to turn all lights off",
			},
		},
		"required": []string{"power"},
	}
	data, _ := json.Marshal(schema)
	return data
}

func buildSetBrightnessSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"light_id": map[string]any{
				"type":        "string",
				"description": "The ID of the light to control",
			},
			"brightness": map[string]any{
				"type":        "integer",
				"description": "Brightness level (0-100)",
				"minimum":     0,
				"maximum":     100,
			},
		},
		"required": []string{"light_id", "brightness"},
	}
	data, _ := json.Marshal(schema)
	return data
}

func buildSetColorSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"light_id": map[string]any{
				"type":        "string",
				"description": "The ID of the light to control",
			},
			"hue": map[string]any{
				"type":        "integer",
				"description": "Hue value (0-360)",
				"minimum":     0,
				"maximum":     360,
			},
			"saturation": map[string]any{
				"type":        "integer",
				"description": "Saturation value (0-100)",
				"minimum":     0,
				"maximum":     100,
			},
			"brightness": map[string]any{
				"type":        "integer",
				"description": "Brightness value (0-100)",
				"minimum":     0,
				"maximum":     100,
			},
			"kelvin": map[string]any{
				"type":        "integer",
				"description": "Color temperature in Kelvin (2500-9000)",
				"minimum":     2500,
				"maximum":     9000,
			},
		},
		"required": []string{"light_id"},
	}
	data, _ := json.Marshal(schema)
	return data
}

func buildListScenesSchema() json.RawMessage {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	data, _ := json.Marshal(schema)
	return data
}

func buildActivateSceneSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scene_id": map[string]any{
				"type":        "string",
				"description": "The ID of the scene to activate",
			},
		},
		"required": []string{"scene_id"},
	}
	data, _ := json.Marshal(schema)
	return data
}
