package main

// Camera represents a viewport that smoothly follows a target or stays fixed.
type Camera struct {
	X, Y        float64
	W, H        int32
	TargetX     float64
	TargetY     float64
	FollowSpeed float64 // 0–1 lerp factor per frame
	Mode        string  // "follow" or "fixed"
	FixedX      float64 // viewport top-left for fixed mode
	FixedY      float64
	initialized bool    // false until first SetTarget / SetFixed
}

// NewCamera creates a camera with the given viewport dimensions.
func NewCamera(w, h int32) *Camera {
	return &Camera{
		W:           w,
		H:           h,
		FollowSpeed: 0.12,
		Mode:        "follow",
	}
}

// SetTarget sets the world position for follow mode.
func (c *Camera) SetTarget(x, y float64) {
	c.TargetX = x
	c.TargetY = y
	if !c.initialized {
		c.initialized = true
		c.X = x - float64(c.W)/2
		c.Y = y - float64(c.H)/2
	}
}

// SetFixed locks the camera centre at a world position (x, y = view centre).
func (c *Camera) SetFixed(x, y float64) {
	c.Mode = "fixed"
	c.FixedX = x - float64(c.W)/2
	c.FixedY = y - float64(c.H)/2
	c.initialized = true
}

// SetFollow switches to player-follow mode.
func (c *Camera) SetFollow() {
	c.Mode = "follow"
}

// Update moves the camera toward its target (follow mode) or snaps to the
// fixed position, then clamps to map bounds.
func (c *Camera) Update(mapPW, mapPH int) {
	if c.Mode == "fixed" {
		c.X = c.FixedX
		c.Y = c.FixedY
	} else {
		desiredX := c.TargetX - float64(c.W)/2
		desiredY := c.TargetY - float64(c.H)/2
		c.X += (desiredX - c.X) * c.FollowSpeed
		c.Y += (desiredY - c.Y) * c.FollowSpeed
	}

	// Clamp to map edges, or centre the map when the viewport is larger.
	if maxX := float64(mapPW) - float64(c.W); maxX < 0 {
		c.X = maxX / 2 // viewport wider than map — centre it
	} else {
		if c.X < 0 {
			c.X = 0
		}
		if c.X > maxX {
			c.X = maxX
		}
	}
	if maxY := float64(mapPH) - float64(c.H); maxY < 0 {
		c.Y = maxY / 2
	} else {
		if c.Y < 0 {
			c.Y = 0
		}
		if c.Y > maxY {
			c.Y = maxY
		}
	}
}

// ScreenX converts a world X to a screen X.
func (c *Camera) ScreenX(worldX float64) float32 {
	return float32(worldX - c.X)
}

// ScreenY converts a world Y to a screen Y.
func (c *Camera) ScreenY(worldY float64) float32 {
	return float32(worldY - c.Y)
}
