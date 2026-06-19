package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

// Player is the player-controlled character.
type Player struct {
	Sprite      *AnimatedSprite
	X, Y        float64 // world position (top-left of bounding box)
	VelX, VelY  float64
	Width       int32 // collision-box width
	Height      int32 // collision-box height
	RenderW     int32 // on-screen sprite width
	RenderH     int32 // on-screen sprite height
	OnGround    bool
	JumpsLeft   int
	FacingRight bool
	JumpHeld    bool // prevents repeated jumps while the key is held
}

// NewPlayer creates a player from an AnimatedSprite at the given world
// position.  renderSize is the on-screen sprite size (match the map tileSize).
func NewPlayer(sprite *AnimatedSprite, x, y float64, renderSize int32) *Player {
	return &Player{
		Sprite:      sprite,
		X:           x,
		Y:           y,
		Width:       PlayerColW,
		Height:      PlayerColH,
		RenderW:     renderSize,
		RenderH:     renderSize,
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
// The sprite is centered on the smaller collision box and bottom-aligned.
func (p *Player) Render(renderer *sdl.Renderer, cam *Camera) {
	rw := float64(p.RenderW)
	rh := float64(p.RenderH)
	rx := float64(cam.ScreenX(p.X)) + (float64(PlayerColW)-rw)/2
	ry := float64(cam.ScreenY(p.Y)) + float64(PlayerColH) - rh

	flip := sdl.FLIP_NONE
	if !p.FacingRight {
		flip = sdl.FLIP_HORIZONTAL
	}

	p.Sprite.Render(renderer, float32(rx), float32(ry), float32(rw), float32(rh), flip, 0)
}

// CenterX returns the world X of the player's center.
func (p *Player) CenterX() float64 {
	return p.X + float64(p.Width)/2
}

// CenterY returns the world Y of the player's center.
func (p *Player) CenterY() float64 {
	return p.Y + float64(p.Height)/2
}

// CheckSpikeHit tests whether the player's collision rectangle overlaps a
// spike tile's triangular danger zone.  The triangle has its apex at
// (centre, 3/4 height) and base spanning the full tile bottom.
func (p *Player) CheckSpikeHit(tileMap *TileMap) bool {
	rect := rectVertices(p.X, p.Y, float64(p.Width), float64(p.Height))
	for _, tc := range tileMap.GetTilesInRect(p.X, p.Y, float64(p.Width), float64(p.Height)) {
		if !tileMap.IsSpike(tc[0], tc[1]) {
			continue
		}
		tx, ty := tileMap.TileToPixel(tc[0], tc[1])
		tw := float64(tileMap.TileWidth)
		th := float64(tileMap.TileHeight)
		rot := tileMap.SpikeRotation(tc[0], tc[1])

		// Build the triangle for the current rotation.
		// Apex is 16 px from the tile edge opposite the base.
		const spikeTip = 16
		var tri [][2]float64
		switch rot {
		case 90:
			// Apex at right edge, base at left edge (full height).
			tri = [][2]float64{
				{tx + tw - spikeTip, ty + th/2}, // apex
				{tx, ty},                         // top-left
				{tx, ty + th},                    // bottom-left
			}
		case 180:
			// Apex near bottom, base at top (ceiling spike).
			tri = [][2]float64{
				{tx + tw/2, ty + th - spikeTip}, // apex
				{tx, ty},                         // top-left
				{tx + tw, ty},                    // top-right
			}
		case 270:
			// Apex at left edge, base at right edge (full height).
			tri = [][2]float64{
				{tx + spikeTip, ty + th/2}, // apex
				{tx + tw, ty},              // top-right
				{tx + tw, ty + th},         // bottom-right
			}
		default:
			// 0° — apex 16 px from top, base at bottom (floor spike).
			tri = [][2]float64{
				{tx + tw/2, ty + spikeTip}, // apex
				{tx, ty + th},              // bottom-left
				{tx + tw, ty + th},         // bottom-right
			}
		}
		if convexPolygonsOverlap(rect, tri) {
			return true
		}
	}
	return false
}

// rectVertices returns the four corners of an AABB.
func rectVertices(x, y, w, h float64) [][2]float64 {
	return [][2]float64{
		{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h},
	}
}

// convexPolygonsOverlap checks two convex polygons for overlap using SAT.
func convexPolygonsOverlap(a, b [][2]float64) bool {
	// Collect all edge normals from both polygons.
	axes := edgeNormals(a)
	axes = append(axes, edgeNormals(b)...)

	for _, axis := range axes {
		minA, maxA := projectPolygon(a, axis)
		minB, maxB := projectPolygon(b, axis)
		if maxA < minB || maxB < minA {
			return false // separating axis found → no overlap
		}
	}
	return true // all axes overlap → polygons intersect
}

// edgeNormals returns perpendicular vectors for each edge of the polygon.
func edgeNormals(verts [][2]float64) [][2]float64 {
	n := len(verts)
	out := make([][2]float64, 0, n)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		dx := verts[j][0] - verts[i][0]
		dy := verts[j][1] - verts[i][1]
		// Perpendicular: (-dy, dx)
		length := dx*dx + dy*dy
		if length == 0 {
			continue
		}
		out = append(out, [2]float64{-dy, dx})
	}
	return out
}

// projectPolygon projects all vertices onto axis, returning (min, max).
func projectPolygon(verts [][2]float64, axis [2]float64) (float64, float64) {
	min := verts[0][0]*axis[0] + verts[0][1]*axis[1]
	max := min
	for _, v := range verts[1:] {
		proj := v[0]*axis[0] + v[1]*axis[1]
		if proj < min {
			min = proj
		}
		if proj > max {
			max = proj
		}
	}
	return min, max
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
