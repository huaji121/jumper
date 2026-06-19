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
	Mode string  `json:"mode"`
	X    float64 `json:"x,omitempty"`
	Y    float64 `json:"y,omitempty"`
}

// TilePatternDef is the value side of the "pattern" map.  It can be either a
// plain string (e.g. "bricks") or a JSON object with type + properties
// (e.g. {"type":"spike","rotation":180}).
type TilePatternDef struct {
	Type     string  `json:"type"`
	Rotation float64 `json:"rotation,omitempty"`
}

// UnmarshalJSON handles both "bricks" and {"type":"spike","rotation":180}.
func (d *TilePatternDef) UnmarshalJSON(b []byte) error {
	// Try plain string first.
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		d.Type = s
		return nil
	}
	// Otherwise decode as object.
	type alias TilePatternDef
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*d = TilePatternDef(a)
	return nil
}

// LevelData is the JSON format for a level file.
type LevelData struct {
	Width       int                        `json:"width"`
	Height      int                        `json:"height"`
	TileSize    int32                      `json:"tileSize"`
	PlayerSpawn Vec2                       `json:"playerSpawn"`
	Camera      CameraConfig               `json:"camera"`
	Zoom        float64                    `json:"zoom"`
	Pattern     map[string]TilePatternDef  `json:"pattern"`
	Tiles       []string                   `json:"tiles"`
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
