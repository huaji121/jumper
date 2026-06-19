package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

// SavePoint is an interactable checkpoint.  When activated (E key while near
// it), it shows an "activated" texture for SavePointActiveMS and becomes the
// player's new spawn point.
type SavePoint struct {
	idleSpr      *AnimatedSprite
	activatedSpr *AnimatedSprite
	X, Y         float64 // world position (top-left)
	W, H         int32
	Activated    bool
	Timer        int64 // ms remaining in activated state
}

// NewSavePoint creates a save point from separate idle and activated sprites.
// renderSize should match the map tileSize so the sprite scales with tiles.
func NewSavePoint(idle, activated *AnimatedSprite, x, y float64, renderSize int32) *SavePoint {
	return &SavePoint{
		idleSpr:      idle,
		activatedSpr: activated,
		X:            x,
		Y:            y,
		W:            renderSize,
		H:            renderSize,
	}
}

// Activate switches to the activated visual and starts the timer.
// Returns true if this was a fresh activation.
func (sp *SavePoint) Activate() bool {
	if sp.Activated {
		return false
	}
	sp.Activated = true
	sp.Timer = SavePointActiveMS
	return true
}

// Update advances the timer and reverts to idle when it expires.
func (sp *SavePoint) Update(dt int64) {
	if sp.Activated {
		sp.activatedSpr.Update(dt)
		sp.Timer -= dt
		if sp.Timer <= 0 {
			sp.Timer = 0
			sp.Activated = false
		}
	} else {
		sp.idleSpr.Update(dt)
	}
}

// Render draws the save point relative to the camera.
func (sp *SavePoint) Render(renderer *sdl.Renderer, cam *Camera) {
	sx := float32(float64(cam.ScreenX(sp.X)))
	sy := float32(float64(cam.ScreenY(sp.Y)))

	spr := sp.idleSpr
	if sp.Activated {
		spr = sp.activatedSpr
	}
	spr.Render(renderer, sx, sy, float32(sp.W), float32(sp.H), sdl.FLIP_NONE)
}

// CenterX returns the world X of the save point's centre.
func (sp *SavePoint) CenterX() float64 { return sp.X + float64(sp.W)/2 }

// CenterY returns the world Y of the save point's centre.
func (sp *SavePoint) CenterY() float64 { return sp.Y + float64(sp.H)/2 }
