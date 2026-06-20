package main

import "github.com/Zyko0/go-sdl3/sdl"

// CollisionType describes how a tile interacts with the player.
type CollisionType int

const (
	CollisionNone  CollisionType = iota // empty / pass-through
	CollisionSolid                      // full AABB
	CollisionSpike                      // triangular danger zone
)

// Tile is anything that can occupy a map cell.
type Tile interface {
	Render(renderer *sdl.Renderer, cam *Camera, x, y float32, w, h int32)
	Update(dt int64)
}

// Collider provides collision behaviour for a tile.
type Collider interface {
	Collision() CollisionType
	Rotation() float64 // degrees for spike orientation
}

// Interactable is a tile the player can interact with (I key).
type Interactable interface {
	OnInteract(g *Game, tileX, tileY int)
}

// --- BrickTile ---

type BrickTile struct{ sprite *AnimatedSprite }

func (t *BrickTile) Render(r *sdl.Renderer, cam *Camera, x, y float32, w, h int32) {
	t.sprite.Render(r, x, y, float32(w), float32(h), sdl.FLIP_NONE, 0)
}
func (t *BrickTile) Update(dt int64)     { t.sprite.Update(dt) }
func (t *BrickTile) Collision() CollisionType { return CollisionSolid }
func (t *BrickTile) Rotation() float64   { return 0 }

// --- SpikeTile ---

type SpikeTile struct {
	sprite   *AnimatedSprite
	rotation float64
}

func (t *SpikeTile) Render(r *sdl.Renderer, cam *Camera, x, y float32, w, h int32) {
	flip := sdl.FLIP_NONE
	angle := 0.0
	if t.rotation == 180 {
		flip = sdl.FLIP_VERTICAL
	} else if t.rotation != 0 {
		angle = t.rotation
	}
	t.sprite.Render(r, x, y, float32(w), float32(h), flip, angle)
}
func (t *SpikeTile) Update(dt int64)     { t.sprite.Update(dt) }
func (t *SpikeTile) Collision() CollisionType { return CollisionSpike }
func (t *SpikeTile) Rotation() float64   { return t.rotation }

// --- SavePointTile ---

type SavePointTile struct {
	idleSpr      *AnimatedSprite
	activatedSpr *AnimatedSprite
	activated    bool
	timer        int64
}

func NewSavePointTile(idle, activated *AnimatedSprite) *SavePointTile {
	return &SavePointTile{idleSpr: idle, activatedSpr: activated}
}

func (t *SavePointTile) Render(r *sdl.Renderer, cam *Camera, x, y float32, w, h int32) {
	spr := t.idleSpr
	if t.activated {
		spr = t.activatedSpr
	}
	spr.Render(r, x, y, float32(w), float32(h), sdl.FLIP_NONE, 0)
}

func (t *SavePointTile) Update(dt int64) {
	if t.activated {
		t.activatedSpr.Update(dt)
		t.timer -= dt
		if t.timer <= 0 {
			t.timer = 0
			t.activated = false
		}
	} else {
		t.idleSpr.Update(dt)
	}
}

func (t *SavePointTile) OnInteract(g *Game, tileX, tileY int) {
	if t.activated {
		return
	}
	t.activated = true
	t.timer = SavePointActiveMS
	ts := int32(TileSize) // spawn relative to tile bottom
	g.SpawnX = float64(tileX)*float64(ts) + float64(ts)/2 - float64(PlayerColW)/2
	g.SpawnY = float64(tileY+1)*float64(ts) - float64(PlayerColH)
}

// --- FlagTile ---

type FlagTile struct{ sprite *AnimatedSprite }

func (t *FlagTile) Render(r *sdl.Renderer, cam *Camera, x, y float32, w, h int32) {
	t.sprite.Render(r, x, y, float32(w), float32(h), sdl.FLIP_NONE, 0)
}
func (t *FlagTile) Update(dt int64) { t.sprite.Update(dt) }
func (t *FlagTile) OnInteract(g *Game, tileX, tileY int) {
	g.showCongrats = true
	g.congratsTimer = 3000
	g.pendingLevel = true
}
