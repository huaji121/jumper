package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

type Player struct {
	Sprite      *AnimatedSprite
	X, Y        float64
	VelX, VelY  float64
	Width       int32
	Height      int32
	RenderW     int32
	RenderH     int32
	OnGround    bool
	JumpsLeft   int
	FacingRight bool
	JumpHeld    bool
	Dead        bool
}

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

func (p *Player) Update(tileMap *TileMap, leftHeld, rightHeld, jumpPressed bool) {
	if p.Dead {
		return
	}
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

	if p.OnGround && p.VelX == 0 {
		p.checkGround(tileMap)
	}

	if p.OnGround {
		p.VelY = 0
	} else {
		p.VelY += Gravity
		if p.VelY > MaxFallSpeed {
			p.VelY = MaxFallSpeed
		}
	}

	if jumpPressed && p.JumpsLeft > 0 && !p.JumpHeld {
		p.VelY = JumpForce
		p.JumpsLeft--
		p.JumpHeld = true
		p.OnGround = false
	}
	if !jumpPressed && p.JumpHeld && p.VelY < 0 {
		p.VelY *= JumpCutFactor
	}
	if !jumpPressed {
		p.JumpHeld = false
	}

	p.Y += p.VelY
	p.resolveY(tileMap)

	if p.VelY >= 0 {
		p.checkGround(tileMap)
	}

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

func (p *Player) checkGround(tileMap *TileMap) {
	footY := p.Y + float64(p.Height)
	checkPoints := []float64{
		p.X + 2,
		p.X + float64(p.Width) - 2,
		p.X + float64(p.Width)/2,
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

func (p *Player) resolveX(tileMap *TileMap) {
	for _, tc := range tileMap.GetTilesInRect(p.X, p.Y, float64(p.Width), float64(p.Height)) {
		if !tileMap.IsSolid(tc[0], tc[1]) {
			continue
		}
		tileLeft, _ := tileMap.TileToPixel(tc[0], tc[1])
		tileRight := tileLeft + float64(tileMap.TileWidth)
		if p.VelX > 0 {
			p.X = tileLeft - float64(p.Width)
		} else if p.VelX < 0 {
			p.X = tileRight
		}
		p.VelX = 0
	}
}

func (p *Player) resolveY(tileMap *TileMap) {
	for _, tc := range tileMap.GetTilesInRect(p.X, p.Y, float64(p.Width), float64(p.Height)) {
		if !tileMap.IsSolid(tc[0], tc[1]) {
			continue
		}
		_, tileTop := tileMap.TileToPixel(tc[0], tc[1])
		tileBottom := tileTop + float64(tileMap.TileHeight)
		if p.VelY > 0 {
			p.Y = tileTop - float64(p.Height)
			p.VelY = 0
			p.OnGround = true
			p.JumpsLeft = MaxJumps
		} else if p.VelY < 0 {
			p.Y = tileBottom
			p.VelY = 0
		}
	}
}

func (p *Player) Render(renderer *sdl.Renderer, cam *Camera) {
	if p.Dead {
		return
	}
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

func (p *Player) CenterX() float64 { return p.X + float64(p.Width)/2 }
func (p *Player) CenterY() float64 { return p.Y + float64(p.Height)/2 }

func (p *Player) Respawn(x, y float64) {
	p.X = x
	p.Y = y
	p.VelX = 0
	p.VelY = 0
	p.OnGround = false
	p.JumpsLeft = MaxJumps
	p.JumpHeld = false
	p.Dead = false
}

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
		const spikeTip = 16
		var tri [][2]float64
		switch rot {
		case 90:
			// Apex at right edge centre, base along left edge.
			tri = [][2]float64{{tx + tw, ty + th/2}, {tx, ty}, {tx, ty + th}}
		case 180:
			// Apex near bottom centre, base along top edge.
			tri = [][2]float64{{tx + tw/2, ty + th - spikeTip}, {tx, ty}, {tx + tw, ty}}
		case 270:
			// Apex at left edge centre, base along right edge.
			tri = [][2]float64{{tx, ty + th/2}, {tx + tw, ty}, {tx + tw, ty + th}}
		default:
			// 0° — apex 16 px from top centre, base along bottom edge.
			tri = [][2]float64{{tx + tw/2, ty + spikeTip}, {tx, ty + th}, {tx + tw, ty + th}}
		}
		if convexPolygonsOverlap(rect, tri) {
			return true
		}
	}
	return false
}

func rectVertices(x, y, w, h float64) [][2]float64 {
	return [][2]float64{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}}
}

func convexPolygonsOverlap(a, b [][2]float64) bool {
	axes := edgeNormals(a)
	axes = append(axes, edgeNormals(b)...)
	for _, axis := range axes {
		minA, maxA := projectPolygon(a, axis)
		minB, maxB := projectPolygon(b, axis)
		if maxA < minB || maxB < minA {
			return false
		}
	}
	return true
}

func edgeNormals(verts [][2]float64) [][2]float64 {
	n := len(verts)
	out := make([][2]float64, 0, n)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		dx := verts[j][0] - verts[i][0]
		dy := verts[j][1] - verts[i][1]
		if dx*dx+dy*dy == 0 {
			continue
		}
		out = append(out, [2]float64{-dy, dx})
	}
	return out
}

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
