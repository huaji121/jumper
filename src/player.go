package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

// Player is the player-controlled character.
type Player struct {
	Sprite      *AnimatedSprite
	X, Y        float64 // world position (top-left of bounding box)
	VelX, VelY  float64
	Width       int32 // bounding-box width
	Height      int32 // bounding-box height
	OnGround    bool
	JumpsLeft   int
	FacingRight bool
	JumpHeld    bool // prevents repeated jumps while the key is held
}

// NewPlayer creates a player from an AnimatedSprite at the given world
// position.  Collision box uses PlayerColW×PlayerColH; rendering is scaled
// to TileSize×TileSize.
func NewPlayer(sprite *AnimatedSprite, x, y float64) *Player {
	return &Player{
		Sprite:      sprite,
		X:           x,
		Y:           y,
		Width:       PlayerColW,
		Height:      PlayerColH,
		JumpsLeft:   MaxJumps,
		FacingRight: true,
	}
}

// Update runs one physics tick.  The game loop calls this at a fixed rate
// (PhysicsDT ms per tick), so the constants are per-tick values — no time
// scaling needed.
func (p *Player) Update(tileMap *TileMap, leftHeld, rightHeld, jumpPressed bool) {
	// --- Horizontal ---
	p.VelX = 0
	if leftHeld {
		p.VelX = -PlayerSpeed
		p.FacingRight = false
	}
	if rightHeld {
		p.VelX = PlayerSpeed
		p.FacingRight = true
	}
	p.X += p.VelX
	p.resolveX(tileMap)

	// If we were on the ground but got pushed sideways, re-check that the
	// ground is still under us (we might have been pushed off a ledge).
	if p.OnGround && p.VelX == 0 {
		p.checkGround(tileMap)
	}

	// --- Vertical ---
	if p.OnGround {
		p.VelY = 0 // stay planted — no gravity drift into the ground
	} else {
		p.VelY += Gravity
		if p.VelY > MaxFallSpeed {
			p.VelY = MaxFallSpeed
		}
	}

	// Jump: full impulse on press.
	if jumpPressed && p.JumpsLeft > 0 && !p.JumpHeld {
		p.VelY = JumpForce
		p.JumpsLeft--
		p.JumpHeld = true
		p.OnGround = false
	}

	// Variable-height jump: releasing the button while ascending cuts the
	// upward speed so a tap gives a short hop and holding gives full height.
	if !jumpPressed && p.JumpHeld && p.VelY < 0 {
		p.VelY *= JumpCutFactor
	}
	if !jumpPressed {
		p.JumpHeld = false
	}

	p.Y += p.VelY
	p.resolveY(tileMap)

	// After moving, check whether we're standing on solid ground.
	// This catches the edge-to-edge case (zero overlap) that resolveY misses,
	// and also detects ground after horizontal-only movement.
	if p.VelY >= 0 {
		p.checkGround(tileMap)
	}

	// --- Animation state ---
	if !p.OnGround {
		if p.VelY < 0 {
			p.Sprite.SetAnimation("jump")
		} else {
			p.Sprite.SetAnimation("fall")
		}
	} else if p.VelX != 0 {
		p.Sprite.SetAnimation("run")
	} else {
		p.Sprite.SetAnimation("idle")
	}

	p.Sprite.Update(PhysicsDT)
}

// checkGround tests whether a solid tile exists directly below the player's
// feet (with a 1px tolerance) and snaps Y / sets OnGround accordingly.
func (p *Player) checkGround(tileMap *TileMap) {
	footY := p.Y + float64(p.Height)

	// Scan a few points along the bottom edge of the collision box.
	checkPoints := []float64{
		p.X + 2,                     // near left edge
		p.X + float64(p.Width) - 2,  // near right edge
		p.X + float64(p.Width)/2,    // centre
	}
	tileRow := int(footY) / int(tileMap.TileHeight)
	if tileRow < 0 || tileRow >= tileMap.Height {
		p.OnGround = false
		return
	}

	for _, checkX := range checkPoints {
		tileCol := int(checkX) / int(tileMap.TileWidth)
		if tileCol < 0 || tileCol >= tileMap.Width {
			continue
		}
		if tileMap.IsSolid(tileCol, tileRow) {
			_, tileTop := tileMap.TileToPixel(tileCol, tileRow)
			p.Y = tileTop - float64(p.Height)
			p.VelY = 0
			p.OnGround = true
			p.JumpsLeft = MaxJumps
			return
		}
	}
	p.OnGround = false
}

// resolveX pushes the player out of solid tiles along the X axis.
func (p *Player) resolveX(tileMap *TileMap) {
	for _, tc := range tileMap.GetTilesInRect(p.X, p.Y, float64(p.Width), float64(p.Height)) {
		if !tileMap.IsSolid(tc[0], tc[1]) {
			continue
		}
		tileLeft, _ := tileMap.TileToPixel(tc[0], tc[1])
		tileRight := tileLeft + float64(tileMap.TileWidth)

		if p.VelX > 0 { // moving right → push to the left of the tile
			p.X = tileLeft - float64(p.Width)
		} else if p.VelX < 0 { // moving left → push to the right of the tile
			p.X = tileRight
		}
		p.VelX = 0
	}
}

// resolveY pushes the player out of solid tiles along the Y axis.
func (p *Player) resolveY(tileMap *TileMap) {
	for _, tc := range tileMap.GetTilesInRect(p.X, p.Y, float64(p.Width), float64(p.Height)) {
		if !tileMap.IsSolid(tc[0], tc[1]) {
			continue
		}
		_, tileTop := tileMap.TileToPixel(tc[0], tc[1])
		tileBottom := tileTop + float64(tileMap.TileHeight)

		if p.VelY > 0 { // falling → land on the tile top
			p.Y = tileTop - float64(p.Height)
			p.VelY = 0
			p.OnGround = true
			p.JumpsLeft = MaxJumps
		} else if p.VelY < 0 { // jumping → hit head on tile bottom
			p.Y = tileBottom
			p.VelY = 0
		}
	}
}

// Render draws the player relative to the camera.
// The sprite (TileSize×TileSize) is centered on the smaller collision box
// (PlayerColW×PlayerColH), with the sprite's bottom edge aligned to the
// collision box bottom (feet on ground).
func (p *Player) Render(renderer *sdl.Renderer, cam *Camera) {
	// Offset the sprite so it is horizontally centered on the collision box
	// and bottom-aligned (feet touch the ground at Y+PlayerColH).
	rx := float64(cam.ScreenX(p.X)) + (float64(PlayerColW)-TileSize)/2
	ry := float64(cam.ScreenY(p.Y)) + float64(PlayerColH) - TileSize

	flip := sdl.FLIP_NONE
	if !p.FacingRight {
		flip = sdl.FLIP_HORIZONTAL
	}

	p.Sprite.Render(renderer, float32(rx), float32(ry), TileSize, TileSize, flip)
}

// CenterX returns the world X of the player's center.
func (p *Player) CenterX() float64 {
	return p.X + float64(p.Width)/2
}

// CenterY returns the world Y of the player's center.
func (p *Player) CenterY() float64 {
	return p.Y + float64(p.Height)/2
}

// Respawn teleports the player to a new position and resets physics state.
func (p *Player) Respawn(x, y float64) {
	p.X = x
	p.Y = y
	p.VelX = 0
	p.VelY = 0
	p.OnGround = false
	p.JumpsLeft = MaxJumps
	p.JumpHeld = false
}
