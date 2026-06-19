package main

import (
	"math"
	"math/rand"

	"github.com/Zyko0/go-sdl3/sdl"
)

// Particle is a single particle with position, velocity, and lifetime.
type Particle struct {
	X, Y        float64
	VelX, VelY  float64
	Lifetime    int64 // ms remaining
	MaxLifetime int64 // ms total (for fade-out)
}

// ParticleSystem manages a collection of particles with a shared sprite.
type ParticleSystem struct {
	Sprite    *AnimatedSprite
	Particles []Particle
}

// NewParticleSystem creates a particle system using the given sprite.
func NewParticleSystem(sprite *AnimatedSprite) *ParticleSystem {
	return &ParticleSystem{Sprite: sprite}
}

// Burst spawns n particles at (x, y) with random velocities and a lifetime
// range.  Each particle gets a random direction and speed.
func (ps *ParticleSystem) Burst(x, y float64, n int, speedMin, speedMax, lifetimeMin, lifetimeMax float64) {
	for i := 0; i < n; i++ {
		angle := rand.Float64() * 2 * 3.1415926535
		speed := speedMin + rand.Float64()*(speedMax-speedMin)
		lifetime := int64(lifetimeMin + rand.Float64()*(lifetimeMax-lifetimeMin))
		ps.Particles = append(ps.Particles, Particle{
			X:           x,
			Y:           y,
			VelX:        speed * math.Cos(angle),
			VelY:        speed * math.Sin(angle) - 2.0, // bias upward
			Lifetime:    lifetime,
			MaxLifetime: lifetime,
		})
	}
}

// Update advances all particles by dt ms and removes dead ones.
func (ps *ParticleSystem) Update(dt int64) {
	alive := ps.Particles[:0]
	for _, p := range ps.Particles {
		p.Lifetime -= dt
		if p.Lifetime <= 0 {
			continue
		}
		p.VelY += 0.15 // gravity
		p.X += float64(p.VelX)
		p.Y += float64(p.VelY)
		alive = append(alive, p)
	}
	ps.Particles = alive
}

// Render draws all particles relative to the camera, fading with lifetime.
func (ps *ParticleSystem) Render(renderer *sdl.Renderer, cam *Camera) {
	for _, p := range ps.Particles {
		alpha := float32(p.Lifetime) / float32(p.MaxLifetime)
		if alpha > 1 {
			alpha = 1
		}
		ps.Sprite.Texture.SetAlphaModFloat(alpha)

		sx := cam.ScreenX(p.X) - float32(ps.Sprite.TexW)/2
		sy := cam.ScreenY(p.Y) - float32(ps.Sprite.TexH)/2
		ps.Sprite.Render(renderer, sx, sy,
			float32(ps.Sprite.TexW), float32(ps.Sprite.TexH), sdl.FLIP_NONE, 0)
	}
	// Reset alpha mod so the sprite texture doesn't affect other rendering.
	ps.Sprite.Texture.SetAlphaModFloat(1.0)
}
