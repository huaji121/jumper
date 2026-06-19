package main

// Camera represents a viewport that smoothly follows a target in the world.
type Camera struct {
	X, Y        float64
	W, H        int32
	TargetX     float64
	TargetY     float64
	FollowSpeed float64 // 0–1 lerp factor per frame
}

// NewCamera creates a camera with the given viewport dimensions.
func NewCamera(w, h int32) *Camera {
	return &Camera{
		W:           w,
		H:           h,
		FollowSpeed: 0.12,
	}
}

// SetTarget sets the world position the camera should follow.
func (c *Camera) SetTarget(x, y float64) {
	c.TargetX = x
	c.TargetY = y
}

// Update moves the camera toward its target and clamps to map bounds.
func (c *Camera) Update(mapPW, mapPH int) {
	desiredX := c.TargetX - float64(c.W)/2
	desiredY := c.TargetY - float64(c.H)/2

	c.X += (desiredX - c.X) * c.FollowSpeed
	c.Y += (desiredY - c.Y) * c.FollowSpeed

	// Clamp
	if c.X < 0 {
		c.X = 0
	}
	if maxX := float64(mapPW) - float64(c.W); c.X > maxX && maxX > 0 {
		c.X = maxX
	}
	if c.Y < 0 {
		c.Y = 0
	}
	if maxY := float64(mapPH) - float64(c.H); c.Y > maxY && maxY > 0 {
		c.Y = maxY
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
