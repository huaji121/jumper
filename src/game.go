package main

import (
	"fmt"
	"math"

	"github.com/Zyko0/go-sdl3/bin/binimg"
	"github.com/Zyko0/go-sdl3/bin/binsdl"
	"github.com/Zyko0/go-sdl3/img"
	"github.com/Zyko0/go-sdl3/sdl"
)

type unloader interface{ Unload() }

type Game struct {
	Window    *sdl.Window
	Renderer  *sdl.Renderer
	Player    *Player
	TileMap   *TileMap
	Camera    *Camera
	Running   bool
	lastTick  uint64
	unloaders []unloader

	Particles *ParticleSystem
	SpawnX    float64
	SpawnY    float64

	interactWasHeld bool
	respawnWasHeld  bool

	congratsSprite *AnimatedSprite
	congratsTex    *sdl.Texture
	showCongrats   bool
	congratsTimer  int64

	playerSprite *AnimatedSprite

	brickSprite  *AnimatedSprite
	spikeSprite  *AnimatedSprite
	flagSprite   *AnimatedSprite
	spTex        *sdl.Texture
	spActTex     *sdl.Texture

	levelPaths   []string
	currentLevel int
	pendingLevel bool
}

func NewGame() (*Game, error) {
	sdlLib := binsdl.Load()
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		sdlLib.Unload()
		return nil, fmt.Errorf("sdl.Init: %w", err)
	}
	imgLib := binimg.Load()

	window, renderer, err := sdl.CreateWindowAndRenderer(
		"Jumper", ScreenWidth, ScreenHeight, 0,
	)
	if err != nil {
		imgLib.Unload()
		sdlLib.Unload()
		sdl.Quit()
		return nil, fmt.Errorf("CreateWindowAndRenderer: %w", err)
	}

	g := &Game{
		Window:   window,
		Renderer: renderer,
		Running:  true,
		lastTick: sdl.Ticks(),
		unloaders: []unloader{sdlLib, imgLib},
	}

	ll, err := LoadLevelsList("assets/levels/levels.json")
	if err != nil {
		g.Cleanup()
		return nil, err
	}
	g.levelPaths = ll.Levels

	if err := g.loadTextures(); err != nil {
		g.Cleanup()
		return nil, err
	}
	if err := g.switchToLevel(0); err != nil {
		g.Cleanup()
		return nil, err
	}
	return g, nil
}

func (g *Game) loadTextures() error {
	t, err := img.LoadTexture(g.Renderer, "assets/textures/bricks.png")
	if err != nil {
		return fmt.Errorf("load bricks.png: %w", err)
	}
	t.SetScaleMode(sdl.SCALEMODE_NEAREST)
	g.brickSprite = NewAnimatedSprite(t)
	g.brickSprite.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
		{X: 0, Y: 0, W: g.brickSprite.TexW, H: g.brickSprite.TexH, Duration: 0},
	}, Loop: true})

	t, err = img.LoadTexture(g.Renderer, "assets/textures/player.png")
	if err != nil {
		return fmt.Errorf("load player.png: %w", err)
	}
	t.SetScaleMode(sdl.SCALEMODE_NEAREST)
	g.playerSprite = NewAnimatedSprite(t)
	ff := func() AnimationFrame {
		return AnimationFrame{X: 0, Y: 0, W: g.playerSprite.TexW, H: g.playerSprite.TexH, Duration: 0}
	}
	g.playerSprite.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{ff()}, Loop: true})
	g.playerSprite.AddAnimation(&Animation{Name: "run", Frames: []AnimationFrame{ff()}, Loop: true})
	g.playerSprite.AddAnimation(&Animation{Name: "jump", Frames: []AnimationFrame{ff()}, Loop: false})
	g.playerSprite.AddAnimation(&Animation{Name: "fall", Frames: []AnimationFrame{ff()}, Loop: false})

	g.spTex, err = img.LoadTexture(g.Renderer, "assets/textures/save_point.png")
	if err != nil {
		return fmt.Errorf("load save_point.png: %w", err)
	}
	g.spTex.SetScaleMode(sdl.SCALEMODE_NEAREST)
	g.spActTex, err = img.LoadTexture(g.Renderer, "assets/textures/save_point_activated.png")
	if err != nil {
		return fmt.Errorf("load save_point_activated.png: %w", err)
	}
	g.spActTex.SetScaleMode(sdl.SCALEMODE_NEAREST)

	t, err = img.LoadTexture(g.Renderer, "assets/textures/spike.png")
	if err != nil {
		return fmt.Errorf("load spike.png: %w", err)
	}
	t.SetScaleMode(sdl.SCALEMODE_NEAREST)
	g.spikeSprite = NewAnimatedSprite(t)
	g.spikeSprite.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
		{X: 0, Y: 0, W: g.spikeSprite.TexW, H: g.spikeSprite.TexH, Duration: 0},
	}, Loop: true})

	t, err = img.LoadTexture(g.Renderer, "assets/textures/flag.png")
	if err != nil {
		return fmt.Errorf("load flag.png: %w", err)
	}
	t.SetScaleMode(sdl.SCALEMODE_NEAREST)
	g.flagSprite = NewAnimatedSprite(t)
	g.flagSprite.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
		{X: 0, Y: 0, W: g.flagSprite.TexW, H: g.flagSprite.TexH, Duration: 0},
	}, Loop: true})

	g.congratsTex, err = img.LoadTexture(g.Renderer, "assets/textures/congratulations.png")
	if err != nil {
		return fmt.Errorf("load congratulations.png: %w", err)
	}
	g.congratsTex.SetScaleMode(sdl.SCALEMODE_NEAREST)
	g.congratsSprite = NewAnimatedSprite(g.congratsTex)
	g.congratsSprite.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
		{X: 0, Y: 0, W: g.congratsSprite.TexW, H: g.congratsSprite.TexH, Duration: 0},
	}, Loop: true})

	t, err = img.LoadTexture(g.Renderer, "assets/textures/blood.png")
	if err != nil {
		return fmt.Errorf("load blood.png: %w", err)
	}
	t.SetScaleMode(sdl.SCALEMODE_NEAREST)
	bloodSprite := NewAnimatedSprite(t)
	bloodSprite.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
		{X: 0, Y: 0, W: bloodSprite.TexW, H: bloodSprite.TexH, Duration: 0},
	}, Loop: true})
	g.Particles = NewParticleSystem(bloodSprite)

	return nil
}

func (g *Game) switchToLevel(idx int) error {
	if idx < 0 || idx >= len(g.levelPaths) {
		return fmt.Errorf("level index %d out of range", idx)
	}
	g.currentLevel = idx
	g.pendingLevel = false

	ld, err := LoadLevel(g.levelPaths[idx])
	if err != nil {
		return err
	}

	tileMap := NewTileMap(ld.Width, ld.Height, ld.TileSize, ld.TileSize)
	ts := ld.TileSize

	for row, line := range ld.Tiles {
		for col, ch := range line {
			pat, ok := ld.Pattern[string(ch)]
			if !ok || pat.Type == "" {
				continue
			}
			switch pat.Type {
			case "bricks":
				tileMap.SetTile(col, row, &BrickTile{sprite: g.brickSprite})
			case "spike":
				tileMap.SetTile(col, row, &SpikeTile{sprite: g.spikeSprite, rotation: pat.Rotation})
			case "flag":
				tileMap.SetTile(col, row, &FlagTile{sprite: g.flagSprite})
			case "save_point":
				idle := NewAnimatedSprite(g.spTex)
				idle.AddAnimation(&Animation{Name: "idle", Frames: []AnimationFrame{
					{X: 0, Y: 0, W: idle.TexW, H: idle.TexH, Duration: 0},
				}, Loop: true})
				act := NewAnimatedSprite(g.spActTex)
				act.AddAnimation(&Animation{Name: "activated", Frames: []AnimationFrame{
					{X: 0, Y: 0, W: act.TexW, H: act.TexH, Duration: 0},
				}, Loop: true})
				tileMap.SetTile(col, row, NewSavePointTile(idle, act))
			}
		}
	}

	player := NewPlayer(g.playerSprite, ld.PlayerSpawn.X, ld.PlayerSpawn.Y, ts)
	g.Renderer.SetVSync(1)

	cam := NewCamera(ScreenWidth, ScreenHeight)
	if ld.Camera.Mode == "fixed" {
		cam.SetFixed(ld.Camera.X, ld.Camera.Y)
	}

	g.TileMap = tileMap
	g.Player = player
	g.Camera = cam
	g.SpawnX = ld.PlayerSpawn.X
	g.SpawnY = ld.PlayerSpawn.Y
	g.showCongrats = false
	g.interactWasHeld = true

	fmt.Printf("[level %d] loaded: %s (spawn: %.0f, %.0f)\n",
		idx+1, g.levelPaths[idx], g.SpawnX, g.SpawnY)

	return nil
}

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

		if g.Camera.Mode != "fixed" {
			g.Camera.SetTarget(g.Player.CenterX(), g.Player.CenterY())
		}
		g.Camera.Update(float64(dt)/1000.0, g.TileMap.PixelWidth(), g.TileMap.PixelHeight())
		g.render()

		elapsed := int64(sdl.Ticks() - now)
		if elapsed < 2 {
			sdl.Delay(1)
		}
	}
	return nil
}

func (g *Game) fixedUpdate() {
	keys := sdl.GetKeyboardState()

	interactKey := keys[sdl.SCANCODE_I]
	respawnKey := keys[sdl.SCANCODE_R]

	if interactKey && !g.interactWasHeld {
		g.interactNearbyTiles()
	}
	g.interactWasHeld = interactKey

	if respawnKey && !g.respawnWasHeld {
		if !g.Player.Dead {
			g.Particles.Burst(g.Player.CenterX(), g.Player.CenterY(), 25, 1.0, 4.0, 500, 1200)
		}
		g.Player.Respawn(g.SpawnX, g.SpawnY)
	}
	g.respawnWasHeld = respawnKey

	if g.showCongrats {
		g.congratsTimer -= PhysicsDT
		if g.congratsTimer <= 0 {
			g.showCongrats = false
			if g.pendingLevel {
				next := g.currentLevel + 1
				if next >= len(g.levelPaths) { next = 0 }
				if err := g.switchToLevel(next); err != nil {
					fmt.Printf("WARNING: level transition failed: %v\n", err)
					g.pendingLevel = false
				}
			}
		}
	}

	left := keys[sdl.SCANCODE_A]
	right := keys[sdl.SCANCODE_D]
	jump := keys[sdl.SCANCODE_J] || keys[sdl.SCANCODE_W] || keys[sdl.SCANCODE_SPACE]
	g.Player.Update(g.TileMap, left, right, jump)

	if !g.Player.Dead {
		if g.Player.CheckSpikeHit(g.TileMap) || g.Player.Y > float64(g.TileMap.PixelHeight())+TileSize {
			g.Player.Dead = true
			g.Particles.Burst(g.Player.CenterX(), g.Player.CenterY(), 25, 1.0, 4.0, 500, 1200)
		}
	}

	g.TileMap.Update(PhysicsDT)
	g.Particles.Update(PhysicsDT)
}

// interactNearbyTiles finds tiles within interaction range and calls their
// OnInteract method.
func (g *Game) interactNearbyTiles() {
	px := g.Player.CenterX()
	py := g.Player.CenterY()
	r := float64(SavePointInteractR)
	ts := float64(g.TileMap.TileWidth)

	for _, tc := range g.TileMap.GetTilesInRect(px-r, py-r, r*2, r*2) {
		t := g.TileMap.GetTile(tc[0], tc[1])
		if it, ok := t.(Interactable); ok {
			tx := float64(tc[0]) * ts
			ty := float64(tc[1]) * ts
			dx := px - (tx + ts/2)
			dy := py - (ty + ts/2)
			if math.Sqrt(dx*dx+dy*dy) <= r {
				it.OnInteract(g, tc[0], tc[1])
				return
			}
		}
	}
}

func (g *Game) render() {
	g.Renderer.SetDrawColor(30, 30, 50, 255)
	g.Renderer.Clear()

	g.TileMap.Render(g.Renderer, g.Camera)
	g.Player.Render(g.Renderer, g.Camera)
	g.Particles.Render(g.Renderer, g.Camera)

	if g.showCongrats {
		cw := float32(g.congratsSprite.TexW)
		ch := float32(g.congratsSprite.TexH)
		cx := float32(g.Camera.W)/2 - cw/2
		cy := float32(g.Camera.H)/2 - ch/2
		g.congratsSprite.Render(g.Renderer, cx, cy, cw, ch, sdl.FLIP_NONE, 0)
	}

	g.Renderer.Present()
}

func (g *Game) Cleanup() {
	if g.Renderer != nil { g.Renderer.Destroy(); g.Renderer = nil }
	if g.Window != nil { g.Window.Destroy(); g.Window = nil }
	sdl.Quit()
	for i := len(g.unloaders) - 1; i >= 0; i-- {
		g.unloaders[i].Unload()
	}
	g.unloaders = nil
}
