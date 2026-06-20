package main

import "math"

type Camera struct {
	X, Y float64
	W, H int32

	Offset     Vector2
	DecaySpeed float64 // 衰减速度，推荐 8~14
	SnapDist   float64 // 吸附阈值，推荐 0.5

	LockX, LockY bool

	Mode             string
	FixedX, FixedY   float64
	TargetX, TargetY float64

	initialized bool
}

type Vector2 struct{ X, Y float64 }

func NewCamera(w, h int32) *Camera {
	return &Camera{
		W:          w,
		H:          h,
		DecaySpeed: 10.0, // 调大更紧跟，调小更飘逸
		SnapDist:   0.5,
		Mode:       "follow",
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

func (c *Camera) Update(dt float64, mapPW, mapPH int) {
	if c.Mode == "fixed" {
		c.X = c.FixedX
		c.Y = c.FixedY
		return
	}

	// 先把 target 限制在地图范围内，再做平滑
	// 这样 Camera 永远追的是合法位置，边界处自然稳定
	rawX := c.TargetX + c.Offset.X - float64(c.W)/2
	rawY := c.TargetY + c.Offset.Y - float64(c.H)/2

	maxX := float64(mapPW) - float64(c.W)
	maxY := float64(mapPH) - float64(c.H)

	var targetX, targetY float64
	if maxX < 0 {
		targetX = maxX / 2
	} else {
		targetX = clamp(rawX, 0, maxX)
	}
	if maxY < 0 {
		targetY = maxY / 2
	} else {
		targetY = clamp(rawY, 0, maxY)
	}

	// 帧率无关的指数衰减
	if !c.LockX {
		c.X = expDecay(c.X, targetX, c.DecaySpeed, dt)
		// 独立轴 snap：消除亚像素拖尾（原 && 改为独立判断）
		if math.Abs(c.X-targetX) < c.SnapDist {
			c.X = targetX
		}
	} else {
		c.X = targetX
	}

	if !c.LockY {
		c.Y = expDecay(c.Y, targetY, c.DecaySpeed, dt)
		if math.Abs(c.Y-targetY) < c.SnapDist {
			c.Y = targetY
		}
	} else {
		c.Y = targetY
	}
}

// 渲染时取整，消除亚像素 ±1px 闪烁
func (c *Camera) ScreenX(worldX float64) float32 {
	return float32(math.Round(worldX - c.X))
}
func (c *Camera) ScreenY(worldY float64) float32 {
	return float32(math.Round(worldY - c.Y))
}

// expDecay：帧率无关指数平滑，t 秒后剩余距离 = 原距离 × e^(-decay×t)
func expDecay(a, b, decay, dt float64) float64 {
	return b + (a-b)*math.Exp(-decay*dt)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
