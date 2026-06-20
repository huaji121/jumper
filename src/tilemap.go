package main

import "github.com/Zyko0/go-sdl3/sdl"

// TileMap is a grid-based map.  Each cell holds a Tile (nil = empty).
// Tiles can implement Collider for physics and Interactable for I-key actions.
type TileMap struct {
	Grid                  [][]Tile
	Width, Height         int
	TileWidth, TileHeight int32
}

func NewTileMap(width, height int, tw, th int32) *TileMap {
	grid := make([][]Tile, height)
	for y := range grid {
		grid[y] = make([]Tile, width)
	}
	return &TileMap{
		Grid:       grid,
		Width:      width,
		Height:     height,
		TileWidth:  tw,
		TileHeight: th,
	}
}

func (tm *TileMap) SetTile(x, y int, t Tile) {
	if x < 0 || x >= tm.Width || y < 0 || y >= tm.Height {
		return
	}
	tm.Grid[y][x] = t
}

func (tm *TileMap) GetTile(x, y int) Tile {
	if x < 0 || x >= tm.Width || y < 0 || y >= tm.Height {
		return nil
	}
	return tm.Grid[y][x]
}

func (tm *TileMap) IsSolid(x, y int) bool {
	t := tm.GetTile(x, y)
	if c, ok := t.(Collider); ok {
		return c.Collision() == CollisionSolid
	}
	return false
}

func (tm *TileMap) IsSpike(x, y int) bool {
	t := tm.GetTile(x, y)
	if c, ok := t.(Collider); ok {
		return c.Collision() == CollisionSpike
	}
	return false
}

func (tm *TileMap) SpikeRotation(x, y int) float64 {
	t := tm.GetTile(x, y)
	if c, ok := t.(Collider); ok && c.Collision() == CollisionSpike {
		return c.Rotation()
	}
	return 0
}

// Update advances every tile's animation by dt ms.
func (tm *TileMap) Update(dt int64) {
	for _, row := range tm.Grid {
		for _, t := range row {
			if t != nil {
				t.Update(dt)
			}
		}
	}
}

// Render draws visible tiles through the camera (frustum culled).
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
			t := tm.Grid[row][col]
			if t == nil {
				continue
			}
			sx := float32(float64(col)*float64(tm.TileWidth) - cam.X)
			sy := float32(float64(row)*float64(tm.TileHeight) - cam.Y)
			t.Render(renderer, cam, sx, sy, tm.TileWidth, tm.TileHeight)
		}
	}
}

func (tm *TileMap) PixelToTile(px, py float64) (int, int) {
	return int(px)/int(tm.TileWidth), int(py)/int(tm.TileHeight)
}

func (tm *TileMap) TileToPixel(tx, ty int) (float64, float64) {
	return float64(tx)*float64(tm.TileWidth), float64(ty)*float64(tm.TileHeight)
}

func (tm *TileMap) GetTilesInRect(rx, ry, rw, rh float64) [][2]int {
	sc := int(rx) / int(tm.TileWidth)
	if sc < 0 { sc = 0 }
	ec := int(rx+rw-1) / int(tm.TileWidth)
	if ec >= tm.Width { ec = tm.Width - 1 }
	sr := int(ry) / int(tm.TileHeight)
	if sr < 0 { sr = 0 }
	er := int(ry+rh-1) / int(tm.TileHeight)
	if er >= tm.Height { er = tm.Height - 1 }
	var out [][2]int
	for row := sr; row <= er; row++ {
		for col := sc; col <= ec; col++ {
			out = append(out, [2]int{col, row})
		}
	}
	return out
}

func (tm *TileMap) PixelWidth() int  { return tm.Width * int(tm.TileWidth) }
func (tm *TileMap) PixelHeight() int { return tm.Height * int(tm.TileHeight) }
