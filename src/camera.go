package main

import (
	"math"
)

type Camera struct {
	X, Y float64
	W, H int32

	// Spring-damper state.
	VelX, VelY float64

	// Tunables.
	Offset         Vector2
	Stiffness      float64 // spring stiffness (higher = snappier)
	Damping        float64 // damping coefficient (higher = less overshoot)
	MaxSpeed       float64 // max camera speed
	DeadZoneRadius float64 // dead zone radius (no spring force inside)
	LockX          bool
	LockY          bool
	StopThreshold  float64 // snap velocity to zero below this
	SnapThreshold  float64 // if distance to target < this, snap to target (prevents sub‑pixel jitter)

	// Mode.
	Mode   string // "follow" or "fixed"
	FixedX float64
	FixedY float64

	// Target for follow mode.
	TargetX float64
	TargetY float64

	initialized bool
}

type Vector2 struct{ X, Y float64 }

func NewCamera(w, h int32) *Camera {
	return &Camera{
		W:              w,
		H:              h,
		Stiffness:      10.0,  // 不变，响应速度
		Damping:        10.0,  // 增加阻尼，快速衰减
		MaxSpeed:       450.0, // 不变
		DeadZoneRadius: 2.5,   // 扩大死区，弹簧在远处才生效
		StopThreshold:  0.1,   // 速度低于0.1直接归零
		SnapThreshold:  2.0,   // 位置距离<2像素时吸附
		Mode:           "follow",
	}
}

func (c *Camera) SetTarget(x, y float64) {
	c.TargetX = x
	c.TargetY = y
	if !c.initialized {
		c.initialized = true
		c.X = x - float64(c.W)/2
		c.Y = y - float64(c.H)/2
	}
}

func (c *Camera) SetFixed(x, y float64) {
	c.Mode = "fixed"
	c.FixedX = x - float64(c.W)/2
	c.FixedY = y - float64(c.H)/2
	c.initialized = true
}

// Update advances the spring-damper simulation by dt seconds.
func (c *Camera) Update(dt float64, mapPW, mapPH int) {
	if c.Mode == "fixed" {
		c.X = c.FixedX
		c.Y = c.FixedY
		c.VelX = 0
		c.VelY = 0
		return
	}

	targetX := c.TargetX + c.Offset.X - float64(c.W)/2
	targetY := c.TargetY + c.Offset.Y - float64(c.H)/2

	dx := targetX - c.X
	dy := targetY - c.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < c.DeadZoneRadius {
		// Inside dead zone: only apply damping, no spring force.
		c.VelX *= 1.0 - c.Damping*dt
		c.VelY *= 1.0 - c.Damping*dt
		vMag := math.Sqrt(c.VelX*c.VelX + c.VelY*c.VelY)
		if vMag < c.StopThreshold {
			c.VelX = 0
			c.VelY = 0
		}
	} else {
		// Spring force: F = stiffness * displacement - damping * velocity.
		fx := dx*c.Stiffness - c.VelX*c.Damping
		fy := dy*c.Stiffness - c.VelY*c.Damping

		// Clamp force magnitude.
		fMag := math.Sqrt(fx*fx + fy*fy)
		maxForce := c.MaxSpeed * 5
		if fMag > maxForce {
			fx = fx / fMag * maxForce
			fy = fy / fMag * maxForce
		}

		c.VelX += fx * dt
		c.VelY += fy * dt

		// Clamp velocity.
		vMag := math.Sqrt(c.VelX*c.VelX + c.VelY*c.VelY)
		if vMag > c.MaxSpeed {
			c.VelX = c.VelX / vMag * c.MaxSpeed
			c.VelY = c.VelY / vMag * c.MaxSpeed
		}
	}

	if c.LockX {
		c.VelX = 0
		c.X = targetX
	}
	if c.LockY {
		c.VelY = 0
		c.Y = targetY
	}

	c.X += c.VelX * dt
	c.Y += c.VelY * dt

	// fmt.Printf("[camera] pos=(%.1f, %.1f) vel=(%.2f, %.2f) target=(%.1f, %.1f)\n",
	// 	c.X, c.Y, c.VelX, c.VelY, c.TargetX, c.TargetY)

	// Clamp to map bounds.
	if maxX := float64(mapPW) - float64(c.W); maxX < 0 {
		c.X = maxX / 2
		c.VelX = 0
	} else {
		if c.X < 0 {
			c.X = 0
			c.VelX = 0
		}
		if c.X > maxX {
			c.X = maxX
			c.VelX = 0
		}
	}
	if maxY := float64(mapPH) - float64(c.H); maxY < 0 {
		c.Y = maxY / 2
		c.VelY = 0
	} else {
		if c.Y < 0 {
			c.Y = 0
			c.VelY = 0
		}
		if c.Y > maxY {
			c.Y = maxY
			c.VelY = 0
		}
	}

	// Snap to target if very close – eliminates sub‑pixel jitter when stopped.
	if math.Abs(c.X-targetX) < c.SnapThreshold && math.Abs(c.Y-targetY) < c.SnapThreshold {
		c.X = targetX
		c.Y = targetY
		c.VelX = 0
		c.VelY = 0
	}
}

func (c *Camera) ScreenX(worldX float64) float32 { return float32(worldX - c.X) }
func (c *Camera) ScreenY(worldY float64) float32 { return float32(worldY - c.Y) }
