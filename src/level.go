package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Vec2 is a simple X/Y position used in level data.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// CameraConfig describes camera behaviour for a level.
type CameraConfig struct {
	Mode string `json:"mode"` // "follow" or "fixed"
	X    float64 `json:"x,omitempty"`
	Y    float64 `json:"y,omitempty"`
}

// LevelData is the JSON format for a level file.
type LevelData struct {
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	TileSize    int32             `json:"tileSize"`
	PlayerSpawn Vec2              `json:"playerSpawn"`
	Camera      CameraConfig      `json:"camera"`
	Pattern     map[string]string `json:"pattern"`
	Tiles       []string          `json:"tiles"`
}

// LoadLevel reads and parses a level JSON file.
func LoadLevel(path string) (*LevelData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var ld LevelData
	if err := json.Unmarshal(data, &ld); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(ld.Tiles) != ld.Height {
		return nil, fmt.Errorf("%s: expected %d tile rows, got %d", path, ld.Height, len(ld.Tiles))
	}
	for i, row := range ld.Tiles {
		if len(row) != ld.Width {
			return nil, fmt.Errorf("%s: row %d expected %d cols, got %d", path, i, ld.Width, len(row))
		}
	}
	return &ld, nil
}
