package main

const (
	ScreenWidth  = 960
	ScreenHeight = 640
	TileSize     = 32

	// Physics — all values are per physics tick (each tick = 1/60 s).
	// The game loop guarantees exactly 60 ticks per in-game second.
	PlayerSpeed  = 2.0  // pixels per tick  (120 px/s → ~8 s across map)
	Gravity      = 0.18 // velocity added per tick
	MaxFallSpeed = 7.0  // terminal velocity
	JumpForce    = -6.0 // initial jump velocity
	MaxJumps     = 2

	// JumpCutFactor is multiplied into VelY when the player releases jump
	// while still ascending.  0.18 means a tap reaches ~1/3 tile (~10% of full).
	JumpCutFactor = 0.18

	// Player collision box — smaller than the rendered sprite so the player
	// can slip through tighter gaps and doesn't feel like a block.
	PlayerColW = 14
	PlayerColH = 22

	// Save point
	SavePointActiveMS  = 500 // ms the "activated" visual lasts
	SavePointInteractR = 48  // px — max distance for E-key interaction

	// Fixed timestep: 1000 ms / 60 Hz ≈ 16 ms per tick.
	PhysicsHz = 60
	PhysicsDT = 1000 / PhysicsHz
	MaxDT     = 200 // clamp accumulated dt to avoid spiral of death (ms)
)
