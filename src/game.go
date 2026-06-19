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
	unloaders []unloader

	SavePoints []*SavePoint
	SpawnX     float64
	SpawnY     float64

	interactWasHeld bool
}

// NewGame initialises SDL, loads assets, builds the level, and wires up the
// player and camera.  Returns a ready-to-run Game, or an error.
func NewGame() (*Game, error) {
	sdlLib := binsdl.Load()
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		sdlLib.Unload()
		return nil, fmt.Errorf("sdl.Init: %w", err)
	}

	imgLib := binimg.Load()

	window, renderer, err := sdl.CreateWindowAndRenderer(
		"Jumper",
		ScreenWidth, ScreenHeight,
		sdl.WINDOW_RESIZABLE,
	)
	if err != nil {
		imgLib.Unload()
		sdlLib.Unload()
		sdl.Quit()
		return nil, fmt.Errorf("CreateWindowAndRenderer: %w", err)
	}

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

	spikeTex, err := img.LoadTexture(renderer, "assets/textures/spike.png")
	if err != nil {
		g.Cleanup()
		return nil, fmt.Errorf("load spike.png: %w", err)
	}
	spikeTex.SetScaleMode(sdl.SCALEMODE_NEAREST)

	// --- Build AnimatedSprites ---

	brickSprite := NewAnimatedSprite(brickTex)
	brickSprite.AddAnimation(&Animation{
		Name: "idle",
		Frames: []AnimationFrame{
			{X: 0, Y: 0, W: brickSprite.TexW, H: brickSprite.TexH, Duration: 0},
		},
		Loop: true,
	})

	spikeSprite := NewAnimatedSprite(spikeTex)
	spikeSprite.AddAnimation(&Animation{
		Name: "idle",
		Frames: []AnimationFrame{
			{X: 0, Y: 0, W: spikeSprite.TexW, H: spikeSprite.TexH, Duration: 0},
		},
		Loop: true,
	})

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

	// --- Load level from JSON ---
	ld, err := LoadLevel("assets/levels/level1.json")
	if err != nil {
		g.Cleanup()
		return nil, err
	}

	tileMap := NewTileMap(ld.Width, ld.Height, ld.TileSize, ld.TileSize)

	// Cache tile-def indices by (type, rotation) so shared defs are reused.
	tileDefCache := map[string]int{}

	getOrCreateDef := func(tag string, def *TileDef) int {
		key := fmt.Sprintf("%s:%v", tag, def.Rotation)
		if idx, ok := tileDefCache[key]; ok {
			return idx
		}
		idx := tileMap.AddDef(def)
		tileDefCache[key] = idx
		return idx
	}

	var savePoints []*SavePoint

	for row, line := range ld.Tiles {
		for col, ch := range line {
			pat, ok := ld.Pattern[string(ch)]
			if !ok || pat.Type == "" {
				continue
			}
			switch pat.Type {
			case "bricks":
				def := &TileDef{Sprite: brickSprite, Solid: true}
				tileMap.SetTile(col, row, getOrCreateDef("brick", def))
			case "spike":
				def := &TileDef{Sprite: spikeSprite, Spike: true, Rotation: pat.Rotation}
				tileMap.SetTile(col, row, getOrCreateDef("spike", def))
			case "save_point":
				idle, act := makeSavePointSprites()
				x := float64(col) * float64(ld.TileSize)
				y := float64(row) * float64(ld.TileSize)
				sp := NewSavePoint(idle, act, x, y, ld.TileSize)
				savePoints = append(savePoints, sp)
			}
		}
	}

	// --- Player ---
	player := NewPlayer(playerSprite, ld.PlayerSpawn.X, ld.PlayerSpawn.Y, TileSize)

	// --- Zoom & presentation ---
	zoom := ld.Zoom
	if zoom <= 0 {
		zoom = 1.0
	}
	logicalW := int32(float64(ScreenWidth) / zoom)
	logicalH := int32(float64(ScreenHeight) / zoom)
	renderer.SetLogicalPresentation(logicalW, logicalH, sdl.LOGICAL_PRESENTATION_OVERSCAN)
	renderer.SetVSync(1)

	cam := NewCamera(logicalW, logicalH)
	if ld.Camera.Mode == "fixed" {
		cam.SetFixed(ld.Camera.X, ld.Camera.Y)
	}

	g.TileMap = tileMap
	g.Player = player
	g.Camera = cam
	g.SavePoints = savePoints
	g.SpawnX = ld.PlayerSpawn.X
	g.SpawnY = ld.PlayerSpawn.Y

	return g, nil
}

// Run executes the main game loop using a fixed-timestep accumulator.
func (g *Game) Run() error {
	var accumulator int64

	for g.Running {
		var event sdl.Event
		for sdl.PollEvent(&event) {
			if event.Type == sdl.EVENT_QUIT {
				g.Running = false
			}
		}
		if !g.Running {
			break
		}

		now := sdl.Ticks()
		dt := int64(now - g.lastTick)
		g.lastTick = now
		if dt > MaxDT {
			dt = MaxDT
		}

		accumulator += dt
		for accumulator >= PhysicsDT {
			g.fixedUpdate()
			accumulator -= PhysicsDT
		}

		g.render()

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

	left := keys[sdl.SCANCODE_A]
	right := keys[sdl.SCANCODE_D]
	jump := keys[sdl.SCANCODE_J] || keys[sdl.SCANCODE_W] || keys[sdl.SCANCODE_SPACE]
	interactKey := keys[sdl.SCANCODE_I]

	if interactKey && !g.interactWasHeld {
		g.interactSavePoints()
	}
	g.interactWasHeld = interactKey

	g.Player.Update(g.TileMap, left, right, jump)

	// Spike collision check — respawn on contact.
	if g.Player.CheckSpikeHit(g.TileMap) {
		g.Player.Respawn(g.SpawnX, g.SpawnY)
	}

	for _, sp := range g.SavePoints {
		sp.Update(PhysicsDT)
	}

	g.TileMap.Update(PhysicsDT)

	if g.Player.Y > float64(g.TileMap.PixelHeight())+TileSize {
		g.Player.Respawn(g.SpawnX, g.SpawnY)
	}

	if g.Camera.Mode != "fixed" {
		g.Camera.SetTarget(g.Player.CenterX(), g.Player.CenterY())
	}
	g.Camera.Update(g.TileMap.PixelWidth(), g.TileMap.PixelHeight())
}

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
				g.SpawnY = sp.Y + float64(sp.H) - float64(PlayerColH)
			}
			return
		}
	}
}

func (g *Game) render() {
	g.Renderer.SetDrawColor(30, 30, 50, 255)
	g.Renderer.Clear()

	g.TileMap.Render(g.Renderer, g.Camera)

	for _, sp := range g.SavePoints {
		sp.Render(g.Renderer, g.Camera)
	}

	g.Player.Render(g.Renderer, g.Camera)

	g.Renderer.Present()
}

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
