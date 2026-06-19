package main

import (
	"fmt"
	"math"

	"github.com/Zyko0/go-sdl3/bin/binimg"
	"github.com/Zyko0/go-sdl3/bin/binsdl"
	"github.com/Zyko0/go-sdl3/img"
	"github.com/Zyko0/go-sdl3/sdl"
)

// unloader is satisfied by both binsdl.library and binimg.library.
type unloader interface{ Unload() }

// Game owns the top-level SDL resources and orchestrates the game loop.
type Game struct {
	Window    *sdl.Window
	Renderer  *sdl.Renderer
	Player    *Player
	TileMap   *TileMap
	Camera    *Camera
	Running   bool
	lastTick  uint64
	unloaders []unloader // native lib handles for cleanup

	SavePoints []*SavePoint
	SpawnX     float64 // current respawn position
	SpawnY     float64

	eWasHeld bool // prevents repeated E-key activations per press
}

// NewGame initialises SDL, loads assets, builds the level, and wires up the
// player and camera.  Returns a ready-to-run Game, or an error.
func NewGame() (*Game, error) {
	// --- SDL native library (embedded DLL extracted to temp) ---
	sdlLib := binsdl.Load()
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		sdlLib.Unload()
		return nil, fmt.Errorf("sdl.Init: %w", err)
	}

	// --- SDL_image native library ---
	imgLib := binimg.Load()

	window, renderer, err := sdl.CreateWindowAndRenderer(
		"Jumper",
		ScreenWidth, ScreenHeight,
		0,
	)
	if err != nil {
		imgLib.Unload()
		sdlLib.Unload()
		sdl.Quit()
		return nil, fmt.Errorf("CreateWindowAndRenderer: %w", err)
	}

	// Enable vsync so the game runs at the display refresh rate.
	renderer.SetVSync(1)

	g := &Game{
		Window:    window,
		Renderer:  renderer,
		Running:   true,
		lastTick:  sdl.Ticks(),
		unloaders: []unloader{sdlLib, imgLib},
	}

	// --- Load textures ---
	brickTex, err := img.LoadTexture(renderer, "assets/textures/bricks.png")
	if err != nil {
		g.Cleanup()
		return nil, fmt.Errorf("load bricks.png: %w", err)
	}
	brickTex.SetScaleMode(sdl.SCALEMODE_NEAREST)

	playerTex, err := img.LoadTexture(renderer, "assets/textures/player.png")
	if err != nil {
		g.Cleanup()
		return nil, fmt.Errorf("load player.png: %w", err)
	}
	playerTex.SetScaleMode(sdl.SCALEMODE_NEAREST)

	spTex, err := img.LoadTexture(renderer, "assets/textures/save_point.png")
	if err != nil {
		g.Cleanup()
		return nil, fmt.Errorf("load save_point.png: %w", err)
	}
	spTex.SetScaleMode(sdl.SCALEMODE_NEAREST)

	spActTex, err := img.LoadTexture(renderer, "assets/textures/save_point_activated.png")
	if err != nil {
		g.Cleanup()
		return nil, fmt.Errorf("load save_point_activated.png: %w", err)
	}
	spActTex.SetScaleMode(sdl.SCALEMODE_NEAREST)

	// --- Build AnimatedSprites ---

	// Brick: a single-frame "idle" animation covering the whole texture.
	brickSprite := NewAnimatedSprite(brickTex)
	brickSprite.AddAnimation(&Animation{
		Name: "idle",
		Frames: []AnimationFrame{
			{X: 0, Y: 0, W: brickSprite.TexW, H: brickSprite.TexH, Duration: 0},
		},
		Loop: true,
	})

	// Player: register every animation state with a full-texture frame.
	playerSprite := NewAnimatedSprite(playerTex)
	fullFrame := func() AnimationFrame {
		return AnimationFrame{
			X: 0, Y: 0,
			W: playerSprite.TexW, H: playerSprite.TexH,
			Duration: 0,
		}
	}
	playerSprite.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{fullFrame()}, Loop: true})
	playerSprite.AddAnimation(&Animation{Name: "run", Frames: []AnimationFrame{fullFrame()}, Loop: true})
	playerSprite.AddAnimation(&Animation{Name: "jump", Frames: []AnimationFrame{fullFrame()}, Loop: false})
	playerSprite.AddAnimation(&Animation{Name: "fall", Frames: []AnimationFrame{fullFrame()}, Loop: false})

	// Save point factory: each gets its own idle + activated sprite pair
	// so animations are independent.
	makeSavePointSprites := func() (*AnimatedSprite, *AnimatedSprite) {
		idle := NewAnimatedSprite(spTex)
		idle.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
			{X: 0, Y: 0, W: idle.TexW, H: idle.TexH, Duration: 0},
		}, Loop: true})

		act := NewAnimatedSprite(spActTex)
		act.AddAnimation(&Animation{Name: "activated", Frames: []AnimationFrame{
			{X: 0, Y: 0, W: act.TexW, H: act.TexH, Duration: 0},
		}, Loop: true})
		return idle, act
	}

	// --- Build tilemap (30×20 tiles at 32 px each → 960×640 world) ---
	tileMap := NewTileMap(30, 20, TileSize, TileSize)
	brickDefIdx := tileMap.AddDef(&TileDef{Sprite: brickSprite, Solid: true})

	// ASCII level layout — X = solid brick, . = empty
	level := []string{
		"..............................",
		"..............................",
		"..............................",
		"..............................",
		"......XXX.....XXX.............",
		"..............................",
		"....XX.......XX..............",
		"..............................",
		"...X...........X....XXX.......",
		"..............................",
		"..XXX.........XXX.............",
		"..............................",
		"...........X..................",
		"..........XXX..............X..",
		".........XXXXX.............X..",
		"......X....................X..",
		"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		"..............................",
		"..............................",
	}
	for row, line := range level {
		for col, ch := range line {
			if ch == 'X' {
				tileMap.SetTile(col, row, brickDefIdx)
			}
		}
	}

	// --- Player ---
	startX := 100.0
	startY := float64(14*int(TileSize) - PlayerColH)
	player := NewPlayer(playerSprite, startX, startY)

	// --- Save points ---
	spIdle1, spAct1 := makeSavePointSprites()
	spIdle2, spAct2 := makeSavePointSprites()
	spIdle3, spAct3 := makeSavePointSprites()
	sp1 := NewSavePoint(spIdle1, spAct1, 10*TileSize, 15*TileSize) // ground, left area
	sp2 := NewSavePoint(spIdle2, spAct2, 20*TileSize, 7*TileSize)  // mid-level platform
	sp3 := NewSavePoint(spIdle3, spAct3, 5*TileSize, 10*TileSize)  // upper platform

	cam := NewCamera(ScreenWidth, ScreenHeight)

	g.TileMap = tileMap
	g.Player = player
	g.Camera = cam
	g.SavePoints = []*SavePoint{sp1, sp2, sp3}
	g.SpawnX = startX
	g.SpawnY = startY

	return g, nil
}

// Run executes the main game loop using a fixed-timestep accumulator.
func (g *Game) Run() error {
	var accumulator int64

	for g.Running {
		// --- Input ---
		var event sdl.Event
		for sdl.PollEvent(&event) {
			if event.Type == sdl.EVENT_QUIT {
				g.Running = false
			}
		}
		if !g.Running {
			break
		}

		// --- Delta time ---
		now := sdl.Ticks()
		dt := int64(now - g.lastTick)
		g.lastTick = now
		if dt > MaxDT {
			dt = MaxDT
		}

		// --- Fixed-timestep physics ---
		accumulator += dt
		for accumulator >= PhysicsDT {
			g.fixedUpdate()
			accumulator -= PhysicsDT
		}

		// --- Render ---
		g.render()

		// Prevent a tight spin loop.
		elapsed := int64(sdl.Ticks() - now)
		if elapsed < 2 {
			sdl.Delay(1)
		}
	}
	return nil
}

// fixedUpdate is called once per physics tick (at PhysicsHz).
func (g *Game) fixedUpdate() {
	keys := sdl.GetKeyboardState()

	left := keys[sdl.SCANCODE_LEFT] || keys[sdl.SCANCODE_A]
	right := keys[sdl.SCANCODE_RIGHT] || keys[sdl.SCANCODE_D]
	jump := keys[sdl.SCANCODE_SPACE] || keys[sdl.SCANCODE_W] || keys[sdl.SCANCODE_UP]
	eKey := keys[sdl.SCANCODE_E]

	// --- E-key interaction (save points) ---
	if eKey && !g.eWasHeld {
		g.interactSavePoints()
	}
	g.eWasHeld = eKey

	// --- Player update ---
	g.Player.Update(g.TileMap, left, right, jump)

	// --- Save point timers ---
	for _, sp := range g.SavePoints {
		sp.Update(PhysicsDT)
	}

	// --- Tile animations ---
	g.TileMap.Update(PhysicsDT)

	// --- Respawn if fell off the map ---
	if g.Player.Y > float64(g.TileMap.PixelHeight())+TileSize {
		g.Player.Respawn(g.SpawnX, g.SpawnY)
	}

	// --- Camera ---
	g.Camera.SetTarget(g.Player.CenterX(), g.Player.CenterY())
	g.Camera.Update(g.TileMap.PixelWidth(), g.TileMap.PixelHeight())
}

// interactSavePoints checks each save point for proximity and activates the
// nearest one that is within interaction range.
func (g *Game) interactSavePoints() {
	px := g.Player.CenterX()
	py := g.Player.CenterY()

	for _, sp := range g.SavePoints {
		dx := px - sp.CenterX()
		dy := py - sp.CenterY()
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= float64(SavePointInteractR) {
			if sp.Activate() {
				g.SpawnX = sp.CenterX() - float64(PlayerColW)/2
				g.SpawnY = sp.Y
			}
			return // only activate the first one in range
		}
	}
}

// render draws the current frame.
func (g *Game) render() {
	g.Renderer.SetDrawColor(30, 30, 50, 255)
	g.Renderer.Clear()

	g.TileMap.Render(g.Renderer, g.Camera)

	// Save points are behind the player.
	for _, sp := range g.SavePoints {
		sp.Render(g.Renderer, g.Camera)
	}

	g.Player.Render(g.Renderer, g.Camera)

	g.Renderer.Present()
}

// Cleanup releases all SDL resources.  Safe to call more than once.
func (g *Game) Cleanup() {
	if g.Renderer != nil {
		g.Renderer.Destroy()
		g.Renderer = nil
	}
	if g.Window != nil {
		g.Window.Destroy()
		g.Window = nil
	}
	sdl.Quit()
	for i := len(g.unloaders) - 1; i >= 0; i-- {
		g.unloaders[i].Unload()
	}
	g.unloaders = nil
}
