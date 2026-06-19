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

// LevelData is the JSON format for a level file.
// Pattern maps single characters to tile types: "b"→"bricks", "x"→"save_point".
// Save points are defined by placing their pattern character in the tile grid.
type LevelData struct {
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	TileSize    int32             `json:"tileSize"`
	PlayerSpawn Vec2              `json:"playerSpawn"`
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
