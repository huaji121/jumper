package main

import (
	"fmt"

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
	// When a real spritesheet is available, replace the frame lists here.
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

	// Place the player above ground level (row 16 is the first ground row).
	player := NewPlayer(playerSprite, 100, float64(14*int(TileSize)-int(playerSprite.TexH)))

	cam := NewCamera(ScreenWidth, ScreenHeight)

	g.TileMap = tileMap
	g.Player = player
	g.Camera = cam

	return g, nil
}

// Run executes the main game loop using a fixed-timestep accumulator.
// Physics always advances in PhysicsDT steps regardless of actual framerate.
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

		// Prevent a tight spin loop: if the frame finished very quickly,
		// yield the CPU briefly so dt stays in a reasonable range.
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

	g.Player.Update(g.TileMap, left, right, jump)
	g.TileMap.Update(PhysicsDT)

	g.Camera.SetTarget(g.Player.CenterX(), g.Player.CenterY())
	g.Camera.Update(g.TileMap.PixelWidth(), g.TileMap.PixelHeight())
}

// render draws the current frame.
func (g *Game) render() {
	g.Renderer.SetDrawColor(30, 30, 50, 255) // dark blue-ish background
	g.Renderer.Clear()

	g.TileMap.Render(g.Renderer, g.Camera)
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
	// Shut down SDL before unloading native libraries (Quit calls into them).
	sdl.Quit()
	for i := len(g.unloaders) - 1; i >= 0; i-- {
		g.unloaders[i].Unload()
	}
	g.unloaders = nil
}
