# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
./run.sh                         # build + run
go build -o jumper.exe ./src/    # build only (always use -o jumper.exe — go build ./src/ alone creates src.exe)
```

The project is a single `main` package in `src/`. Dependencies are managed by `go.mod`. No tests or linters are configured.

## Dependencies

- **Renderer**: `github.com/Zyko0/go-sdl3` (SDL3 Go bindings — native DLLs are embedded via `binsdl`/`binimg`, no system SDL install required)
- All textures use `SetScaleMode(sdl.SCALEMODE_NEAREST)` — pixel-art nearest-neighbour filtering.

## Architecture

### Game loop (fixed-timestep)

`game.go:Run()` uses an accumulator pattern: physics always ticks at exactly `PhysicsHz` (60 Hz, `PhysicsDT` = 16 ms per tick) regardless of actual framerate. `fixedUpdate()` is called once per physics tick; rendering happens as often as possible. VSync is enabled. Window is fixed at 1280×720 (not resizable).

### Entity design

- **AnimatedSprite** — wraps an `*sdl.Texture` + map of named `Animation`s. Each `Animation` holds `[]AnimationFrame` (sub-rect + duration in ms). `SetAnimation(name)` switches states; `Update(dt)` advances the clock. Used by Player, TileDef, and ParticleSystem.
- **Tile interface** — unified abstraction for all map cells (`tile.go`). `Tile.Render()` / `Tile.Update()` for visuals; `Collider` interface for collision (`CollisionSolid` / `CollisionSpike`); `Interactable` interface for I-key actions. Implementations: `BrickTile`, `SpikeTile`, `SavePointTile`, `FlagTile`.
- **ParticleSystem** — manages a pool of `Particle`s with velocity/lifetime. `Burst(x, y, n, …)` spawns n particles at a world position with random directions and speeds. Particles fade out (alpha based on remaining lifetime) and are removed when expired. Used for blood splatter on death.
- **Player** — separate collision box (`PlayerColW`/`PlayerColH`, 20×26) and render size (passed at construction, defaults to `TileSize`=32). Physics (gravity, variable-height jump, double-jump) live in `Update()`. Ground detection uses `checkGround()` which explicitly scans tiles below the player's feet — this is the primary ground check, not overlap-based collision. `resolveX`/`resolveY` handle overlap ejection as a backup. On death (spike or fall) the `Dead` flag is set and blood particles burst — press **R** to respawn.
- **TileMap** — grid of `Tile` interfaces. Each tile type implements `Tile` (Render/Update) and optionally `Collider` (Collision/Rotation) or `Interactable` (OnInteract). `BrickTile`/`SpikeTile`/`SavePointTile`/`FlagTile` all share this unified system. Renders only visible tiles (camera-frustum culled). `GetTilesInRect()` for AABB queries.
- **SavePoint** — implement `Tile` + `Interactable`. Two `AnimatedSprite`s (idle / activated). Activated on I key within `SavePointInteractR` radius. Timer expires after `SavePointActiveMS` ms.
- **Camera** — two modes: `"follow"` (spring-damper physics toward player centre: `force = stiffness*dx - damping*velocity`) and `"fixed"` (snaps to configured world position). Supports dead zone, max speed, `LockX`/`LockY`, offset, and stop threshold. Updated every frame with real dt for smooth, physics-based scrolling.

### Key physics details (constants.go)

All physics values are **per tick** (1/60 s). No time-scaling in the physics methods — the fixed-timestep guarantees consistent tick rate.

| Constant | Value | Meaning |
|---|---|---|
| `PlayerSpeed` | 2.0 | px/tick horizontal |
| `Gravity` | 0.18 | px/tick² downward |
| `JumpForce` | -6.0 | initial upward velocity |
| `JumpCutFactor` | 0.18 | multiplier when releasing jump while ascending (tap ≈ 1/3 tile, hold ≈ 3 tiles) |
| `MaxFallSpeed` | 7.0 | terminal velocity |
| `MaxJumps` | 2 | double-jump enabled |

### Level format (assets/levels/*.json)

```json
{
  "width": 60, "height": 30, "tileSize": 32,
  "playerSpawn": { "x": 100, "y": 902 },
  "camera": { "mode": "follow" },
  "pattern": { ".": "", "b": "bricks", "x": "save_point" },
  "tiles": ["....bbb....", ...]
}
```

- `pattern` maps characters to tile types. `""` = empty, `"bricks"` = solid tile, `"save_point"` = save point entity (non-solid, placed at that grid position).
- `camera.mode`: `"follow"` or `"fixed"` (with `"x"`/`"y"` for view centre).
- `width` × `height` must match the `tiles` array (rows = height, each row = width chars). Validation fails with a clear error on mismatch.

### Controls (hardcoded in game.go:fixedUpdate)

- Move: **A** / **D**
- Jump: **J** / **W** / **Space**
- Interact (save points / flags): **I**
- Respawn (after death): **R**
- Quit: close window

### Textures

Located in `assets/textures/`. All are very small pixel-art (e.g. player is 6×11 px native). They are scaled up at render time by the `renderSize` parameter. To add multi-frame animations, replace the single-frame `Animation` lists with actual spritesheet sub-rects and durations.

### Cleanup order

`Game.Cleanup()` calls: `Renderer.Destroy()` → `Window.Destroy()` → `sdl.Quit()` → native library `Unload()` (reverse order). SDL must be shut down before unloading the native DLLs.
