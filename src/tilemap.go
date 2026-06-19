package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

// TileDef defines a tile type: its animated sprite and collision flag.
// Multiple grid cells can share the same TileDef instance.
type TileDef struct {
	Sprite *AnimatedSprite
	Solid  bool
}

// TileMap is a grid-based map supporting animated tiles.
// Cells hold indices into Defs: -1 is empty, 0+ is a Defs index.
type TileMap struct {
	Defs                  []*TileDef
	Grid                  [][]int
	Width, Height         int   // grid dimensions in tiles
	TileWidth, TileHeight int32 // pixel size of one tile
}

// NewTileMap creates an empty tilemap with every cell set to -1 (empty).
func NewTileMap(width, height int, tileWidth, tileHeight int32) *TileMap {
	grid := make([][]int, height)
	for y := range grid {
		grid[y] = make([]int, width)
		for x := range grid[y] {
			grid[y][x] = -1
		}
	}
	return &TileMap{
		Grid:       grid,
		Width:      width,
		Height:     height,
		TileWidth:  tileWidth,
		TileHeight: tileHeight,
	}
}

// AddDef registers a tile definition and returns its index.
func (tm *TileMap) AddDef(def *TileDef) int {
	tm.Defs = append(tm.Defs, def)
	return len(tm.Defs) - 1
}

// SetTile assigns a tile definition index to a grid cell (-1 for empty).
func (tm *TileMap) SetTile(x, y, defIndex int) {
	if x < 0 || x >= tm.Width || y < 0 || y >= tm.Height {
		return
	}
	tm.Grid[y][x] = defIndex
}

// GetTile returns the tile definition index at a cell, or -1 if empty/OOB.
func (tm *TileMap) GetTile(x, y int) int {
	if x < 0 || x >= tm.Width || y < 0 || y >= tm.Height {
		return -1
	}
	return tm.Grid[y][x]
}

// IsSolid reports whether the tile at (x, y) exists and is solid.
func (tm *TileMap) IsSolid(x, y int) bool {
	idx := tm.GetTile(x, y)
	if idx < 0 || idx >= len(tm.Defs) {
		return false
	}
	return tm.Defs[idx].Solid
}

// Update advances all tile sprite animations by dt ms.
func (tm *TileMap) Update(dt int64) {
	for _, def := range tm.Defs {
		if def.Sprite != nil {
			def.Sprite.Update(dt)
		}
	}
}

// Render draws the visible portion of the tilemap through the camera.
func (tm *TileMap) Render(renderer *sdl.Renderer, cam *Camera) {
	startCol := int(cam.X) / int(tm.TileWidth)
	if startCol < 0 {
		startCol = 0
	}
	endCol := int(cam.X+float64(cam.W))/int(tm.TileWidth) + 1
	if endCol > tm.Width {
		endCol = tm.Width
	}
	startRow := int(cam.Y) / int(tm.TileHeight)
	if startRow < 0 {
		startRow = 0
	}
	endRow := int(cam.Y+float64(cam.H))/int(tm.TileHeight) + 1
	if endRow > tm.Height {
		endRow = tm.Height
	}

	for row := startRow; row < endRow; row++ {
		for col := startCol; col < endCol; col++ {
			idx := tm.Grid[row][col]
			if idx < 0 || idx >= len(tm.Defs) {
				continue
			}
			def := tm.Defs[idx]
			if def.Sprite == nil {
				continue
			}
			sx := float32(float64(col)*float64(tm.TileWidth) - cam.X)
			sy := float32(float64(row)*float64(tm.TileHeight) - cam.Y)
			def.Sprite.Render(renderer, sx, sy,
				float32(tm.TileWidth), float32(tm.TileHeight), sdl.FLIP_NONE)
		}
	}
}

// PixelToTile converts world pixel coordinates to tile grid coordinates.
func (tm *TileMap) PixelToTile(px, py float64) (int, int) {
	return int(px) / int(tm.TileWidth), int(py) / int(tm.TileHeight)
}

// TileToPixel converts tile grid coordinates to world pixel coordinates
// (top-left corner of the tile).
func (tm *TileMap) TileToPixel(tx, ty int) (float64, float64) {
	return float64(tx) * float64(tm.TileWidth), float64(ty) * float64(tm.TileHeight)
}

// GetTilesInRect returns tile grid coordinates that overlap the given AABB.
func (tm *TileMap) GetTilesInRect(rx, ry, rw, rh float64) [][2]int {
	startCol := int(rx) / int(tm.TileWidth)
	if startCol < 0 {
		startCol = 0
	}
	endCol := int(rx+rw-1) / int(tm.TileWidth)
	if endCol >= tm.Width {
		endCol = tm.Width - 1
	}
	startRow := int(ry) / int(tm.TileHeight)
	if startRow < 0 {
		startRow = 0
	}
	endRow := int(ry+rh-1) / int(tm.TileHeight)
	if endRow >= tm.Height {
		endRow = tm.Height - 1
	}

	var result [][2]int
	for row := startRow; row <= endRow; row++ {
		for col := startCol; col <= endCol; col++ {
			result = append(result, [2]int{col, row})
		}
	}
	return result
}

// PixelWidth returns the total map width in pixels.
func (tm *TileMap) PixelWidth() int {
	return tm.Width * int(tm.TileWidth)
}

// PixelHeight returns the total map height in pixels.
func (tm *TileMap) PixelHeight() int {
	return tm.Height * int(tm.TileHeight)
}
