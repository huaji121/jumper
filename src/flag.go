package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

// Flag is a level-completion marker.  Interacting with it (I key) shows a
// congratulations overlay centred on screen.
type Flag struct {
	Sprite *AnimatedSprite
	X, Y   float64
	W, H   int32
}

// NewFlag creates a flag from an AnimatedSprite at a world position.
func NewFlag(sprite *AnimatedSprite, x, y float64, renderSize int32) *Flag {
	return &Flag{
		Sprite: sprite,
		X:      x,
		Y:      y,
		W:      renderSize,
		H:      renderSize,
	}
}

// Render draws the flag relative to the camera.
func (f *Flag) Render(renderer *sdl.Renderer, cam *Camera) {
	sx := float32(float64(cam.ScreenX(f.X)))
	sy := float32(float64(cam.ScreenY(f.Y)))
	f.Sprite.Render(renderer, sx, sy, float32(f.W), float32(f.H), sdl.FLIP_NONE, 0)
}

// CenterX returns the world X of the flag's centre.
func (f *Flag) CenterX() float64 { return f.X + float64(f.W)/2 }

// CenterY returns the world Y of the flag's centre.
func (f *Flag) CenterY() float64 { return f.Y + float64(f.H)/2 }
