package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Zyko0/go-sdl3/sdl"
)

type Editor struct {
	Enabled      bool
	CameraSpeed  float64
	CrossSprite  *AnimatedSprite
	CurrentTile  string // pattern key to place
	MouseLeft    bool
	MouseRight   bool
	MouseX, MouseY float32
}

func NewEditor(cross *AnimatedSprite) *Editor {
	return &Editor{
		CameraSpeed: 5,
		CrossSprite: cross,
		CurrentTile: "b",
	}
}

// Update moves the camera with WASD when editing.
func (ed *Editor) Update(cam *Camera, keys []bool) {
	if !ed.Enabled {
		return
	}
	s := ed.CameraSpeed
	if keys[sdl.SCANCODE_W] {
		cam.Y -= s
	}
	if keys[sdl.SCANCODE_S] {
		cam.Y += s
	}
	if keys[sdl.SCANCODE_A] {
		cam.X -= s
	}
	if keys[sdl.SCANCODE_D] {
		cam.X += s
	}
}

// HandleMouse processes left/right clicks for tile placement/removal.
func (ed *Editor) HandleMouse(g *Game) {
	if !ed.Enabled {
		return
	}
	tileMap := g.TileMap
	cam := g.Camera

	// Mouse screen position → world position.
	wx := cam.X + float64(ed.MouseX)
	wy := cam.Y + float64(ed.MouseY)
	col := int(wx) / int(tileMap.TileWidth)
	row := int(wy) / int(tileMap.TileHeight)

	if col < 0 || col >= tileMap.Width || row < 0 || row >= tileMap.Height {
		return
	}

	if ed.MouseLeft {
		ed.placeTile(g, col, row)
		ed.MouseLeft = false
	}
	if ed.MouseRight {
		tileMap.SetTile(col, row, nil)
		ed.MouseRight = false
	}
}

func (ed *Editor) placeTile(g *Game, col, row int) {
	ld := &g.levelData // need to store levelData on Game
	_ = ld
	pat, ok := g.levelData.Pattern[ed.CurrentTile]
	if !ok || pat.Type == "" {
		return
	}
	switch pat.Type {
	case "bricks":
		g.TileMap.SetTile(col, row, &BrickTile{sprite: g.brickSprite})
	case "spike":
		g.TileMap.SetTile(col, row, &SpikeTile{sprite: g.spikeSprite, rotation: pat.Rotation})
	case "flag":
		g.TileMap.SetTile(col, row, &FlagTile{sprite: g.flagSprite})
	case "save_point":
		idle := NewAnimatedSprite(g.spTex)
		idle.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
			{X: 0, Y: 0, W: idle.TexW, H: idle.TexH, Duration: 0},
		}, Loop: true})
		act := NewAnimatedSprite(g.spActTex)
		act.AddAnimation(&Animation{Name: "activated", Frames: []AnimationFrame{
			{X: 0, Y: 0, W: act.TexW, H: act.TexH, Duration: 0},
		}, Loop: true})
		g.TileMap.SetTile(col, row, NewSavePointTile(idle, act))
	}
}

// Render draws the cross hair at screen centre when editing.
func (ed *Editor) Render(renderer *sdl.Renderer, cam *Camera) {
	if !ed.Enabled {
		return
	}
	cw := float32(ed.CrossSprite.TexW)
	ch := float32(ed.CrossSprite.TexH)
	cx := float32(cam.W)/2 - cw/2
	cy := float32(cam.H)/2 - ch/2
	ed.CrossSprite.Render(renderer, cx, cy, cw, ch, sdl.FLIP_NONE, 0)
}

// HandleCommand dispatches edit commands from the console.
func (ed *Editor) HandleCommand(g *Game, input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "mode":
		if len(parts) < 2 {
			return
		}
		switch parts[1] {
		case "edit":
			ed.Enabled = true
			g.Console.Add("[editor] mode: edit")
		case "game":
			ed.Enabled = false
			g.Player.Respawn(g.SpawnX, g.SpawnY)
			g.Console.Add("[editor] mode: game")
		}

	case "edit":
		if len(parts) < 2 {
			return
		}
		switch parts[1] {
		case "tile":
			ed.cmdTile(g, parts[2:])
		case "level":
			ed.cmdLevel(g, parts[2:])
		case "cameramode":
			ed.cmdCamera(g, parts[2:])
		case "spawnpoint":
			wx := g.Camera.X + float64(g.Camera.W)/2
			wy := g.Camera.Y + float64(g.Camera.H)/2
			g.SpawnX = wx - float64(PlayerColW)/2
			g.SpawnY = wy
			g.levelData.PlayerSpawn.X = g.SpawnX
			g.levelData.PlayerSpawn.Y = g.SpawnY
			g.Console.Add(fmt.Sprintf("[editor] spawn set to (%.0f, %.0f)", g.SpawnX, g.SpawnY))
		}
	}
}

func (ed *Editor) cmdTile(g *Game, args []string) {
	if len(args) < 1 {
		return
	}
	switch args[0] {
	case "new":
		// edit tile new "key":"type" or edit tile new "key":{"type":"spike","rotation":90}
		if len(args) < 2 {
			return
		}
		// Reconstruct the JSON argument (may contain spaces).
		jsonStr := strings.Join(args[1:], " ")
		// Parse "key":value
		colon := strings.Index(jsonStr, ":")
		if colon < 0 {
			return
		}
		key := strings.Trim(jsonStr[:colon], `" `)
		valStr := strings.TrimSpace(jsonStr[colon+1:])

		var def TilePatternDef
		if strings.HasPrefix(valStr, "{") {
			if err := json.Unmarshal([]byte(valStr), &def); err != nil {
				g.Console.Add(fmt.Sprintf("[editor] parse error: %v", err))
				return
			}
		} else {
			def.Type = strings.Trim(valStr, `" `)
		}
		g.levelData.Pattern[key] = def
		g.Console.Add(fmt.Sprintf("[editor] pattern %q added: type=%s rotation=%.0f", key, def.Type, def.Rotation))

	case "set":
		if len(args) < 2 {
			return
		}
		key := args[1]
		if _, ok := g.levelData.Pattern[key]; ok {
			ed.CurrentTile = key
			g.Console.Add(fmt.Sprintf("[editor] current tile = %q", key))
		} else {
			g.Console.Add(fmt.Sprintf("[editor] pattern %q not found", key))
		}

	case "list":
		g.Console.Add("[editor] patterns:")
		for k, v := range g.levelData.Pattern {
			if v.Rotation != 0 {
				g.Console.Add(fmt.Sprintf("  %q: type=%s rotation=%.0f", k, v.Type, v.Rotation))
			} else if v.Type != "" {
				g.Console.Add(fmt.Sprintf("  %q: %s", k, v.Type))
			}
		}
	}
}

func (ed *Editor) cmdLevel(g *Game, args []string) {
	if len(args) < 1 {
		return
	}
	switch args[0] {
	case "save":
		// Serialise modified level data back to JSON.
		ld := g.levelData
		ts := ld.TileSize
		// Rebuild tiles array from current tilemap.
		tiles := make([]string, ld.Height)
		for row := 0; row < ld.Height; row++ {
			line := make([]byte, ld.Width)
			for col := 0; col < ld.Width; col++ {
				t := g.TileMap.GetTile(col, row)
				if t == nil {
					line[col] = '.'
					continue
				}
				// Find the pattern key for this tile.
				found := false
				for k, v := range ld.Pattern {
					if len(k) == 0 || v.Type == "" {
						continue
					}
					switch v.Type {
					case "bricks":
						if _, ok := t.(*BrickTile); ok {
							line[col] = k[0]
							found = true
						}
					case "spike":
						if st, ok := t.(*SpikeTile); ok && st.rotation == v.Rotation {
							line[col] = k[0]
							found = true
						}
					case "save_point":
						if _, ok := t.(*SavePointTile); ok {
							line[col] = k[0]
							found = true
						}
					case "flag":
						if _, ok := t.(*FlagTile); ok {
							line[col] = k[0]
							found = true
						}
					}
					if found {
						break
					}
				}
				if !found {
					line[col] = '.'
				}
			}
			tiles[row] = string(line)
		}
		ld.Tiles = tiles
		_ = ts

		// Write to file.
		path := g.levelPaths[g.currentLevel]
		data, err := json.MarshalIndent(ld, "", "  ")
		if err != nil {
			g.Console.Add(fmt.Sprintf("[editor] save error: %v", err))
			return
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			g.Console.Add(fmt.Sprintf("[editor] save error: %v", err))
			return
		}
		g.Console.Add(fmt.Sprintf("[editor] saved: %s", path))

	case "change":
		if len(args) < 2 {
			return
		}
		idx, err := strconv.Atoi(args[1])
		if err != nil {
			g.Console.Add(fmt.Sprintf("[editor] invalid level index: %s", args[1]))
			return
		}
		idx-- // 1-based to 0-based
		if err := g.switchToLevel(idx); err != nil {
			g.Console.Add(fmt.Sprintf("[editor] %v", err))
		}
	}
}

func (ed *Editor) cmdCamera(g *Game, args []string) {
	if len(args) < 1 {
		return
	}
	switch args[0] {
	case "follow":
		g.Camera.Mode = "follow"
		g.Console.Add("[editor] camera mode: follow")
	case "fixed":
		if len(args) < 3 {
			g.Console.Add("[editor] usage: edit cameramode fixed X Y")
			return
		}
		x, _ := strconv.ParseFloat(args[1], 64)
		y, _ := strconv.ParseFloat(args[2], 64)
		g.Camera.SetFixed(x, y)
		g.Console.Add(fmt.Sprintf("[editor] camera mode: fixed (%.0f, %.0f)", x, y))
	}
}
