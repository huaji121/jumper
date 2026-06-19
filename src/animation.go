package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

// AnimationFrame describes one frame within a spritesheet.
type AnimationFrame struct {
	X, Y, W, H float32 // region within the texture (pixels)
	Duration   int64   // milliseconds this frame is shown
}

// Animation is a named sequence of frames.
type Animation struct {
	Name   string
	Frames []AnimationFrame
	Loop   bool
}

// AnimatedSprite wraps an SDL texture and supports multiple named animations.
// The texture is treated as a spritesheet: each animation's frames are
// rectangular sub-regions.  For a single-frame image, register one Animation
// containing one frame that covers the whole texture.
type AnimatedSprite struct {
	Texture     *sdl.Texture
	Animations  map[string]*Animation
	CurrentAnim string
	FrameIndex  int
	Elapsed     int64 // ms accumulated in the current frame
	Paused      bool
	TexW        float32 // cached from texture.W
	TexH        float32 // cached from texture.H
}

// NewAnimatedSprite creates an AnimatedSprite wrapping an already-loaded SDL
// texture.  The caller retains ownership of the texture (Destroy when done).
func NewAnimatedSprite(texture *sdl.Texture) *AnimatedSprite {
	return &AnimatedSprite{
		Texture:    texture,
		Animations: make(map[string]*Animation),
		TexW:       float32(texture.W),
		TexH:       float32(texture.H),
	}
}

// AddAnimation registers a named animation.  If it is the first animation
// added it automatically becomes the current one.
func (s *AnimatedSprite) AddAnimation(anim *Animation) {
	s.Animations[anim.Name] = anim
	if s.CurrentAnim == "" {
		s.CurrentAnim = anim.Name
	}
}

// SetAnimation switches to a named animation, resetting to frame 0.
// If the animation is already playing the call is a no-op.
func (s *AnimatedSprite) SetAnimation(name string) {
	if _, ok := s.Animations[name]; !ok {
		return
	}
	if s.CurrentAnim == name {
		return
	}
	s.CurrentAnim = name
	s.FrameIndex = 0
	s.Elapsed = 0
}

// Update advances the animation clock by dt milliseconds.
func (s *AnimatedSprite) Update(dt int64) {
	if s.Paused {
		return
	}
	anim := s.Animations[s.CurrentAnim]
	if anim == nil || len(anim.Frames) == 0 {
		return
	}

	s.Elapsed += dt
	frameDur := anim.Frames[s.FrameIndex].Duration
	if frameDur <= 0 {
		frameDur = 100 // default 100 ms per frame
	}

	for s.Elapsed >= frameDur {
		s.Elapsed -= frameDur
		if s.FrameIndex+1 < len(anim.Frames) {
			s.FrameIndex++
		} else if anim.Loop {
			s.FrameIndex = 0
		}
		// refresh duration for the (possibly new) frame
		frameDur = anim.Frames[s.FrameIndex].Duration
		if frameDur <= 0 {
			frameDur = 100
		}
	}
}

// CurrentFrame returns the texture region for the active frame.
func (s *AnimatedSprite) CurrentFrame() *sdl.FRect {
	anim := s.Animations[s.CurrentAnim]
	if anim == nil || len(anim.Frames) == 0 {
		return &sdl.FRect{X: 0, Y: 0, W: s.TexW, H: s.TexH}
	}
	f := anim.Frames[s.FrameIndex]
	return &sdl.FRect{X: f.X, Y: f.Y, W: f.W, H: f.H}
}

// Render draws the sprite at the given screen position and size.
// flip controls horizontal/vertical mirroring; angle rotates around the
// sprite centre (degrees, counter-clockwise).
func (s *AnimatedSprite) Render(renderer *sdl.Renderer, x, y, w, h float32, flip sdl.FlipMode, angle float64) {
	src := s.CurrentFrame()
	dst := &sdl.FRect{X: x, Y: y, W: w, H: h}
	if flip == sdl.FLIP_NONE && angle == 0 {
		renderer.RenderTexture(s.Texture, src, dst)
	} else {
		center := &sdl.FPoint{X: w / 2, Y: h / 2}
		renderer.RenderTextureRotated(s.Texture, src, dst, angle, center, flip)
	}
}
