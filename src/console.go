package main

import "github.com/Zyko0/go-sdl3/sdl"

type Console struct {
	Visible    bool
	Messages   []string
	Input      string
	ScrollPos  int
	MaxVisible int
}

func NewConsole() *Console {
	return &Console{MaxVisible: 10}
}

func (c *Console) Add(msg string) {
	c.Messages = append(c.Messages, msg)
}

func (c *Console) HandleEvent(event *sdl.Event) {
	if !c.Visible {
		if event.Type == sdl.EVENT_KEY_DOWN {
			kev := event.KeyboardEvent()
			if kev.Scancode == sdl.SCANCODE_SLASH && !kev.Repeat {
				c.Visible = true
				c.Input = ""
				c.ScrollPos = 0
			}
		}
		return
	}

	// Console is open.
	if event.Type == sdl.EVENT_KEY_DOWN {
		kev := event.KeyboardEvent()
		sc := kev.Scancode

		if sc == sdl.SCANCODE_ESCAPE {
			c.Visible = false
			return
		}
		if sc == sdl.SCANCODE_RETURN || sc == sdl.SCANCODE_KP_ENTER {
			if c.Input != "" {
				c.Add("> " + c.Input)
				c.Input = ""
				c.ScrollPos = 0
			}
			return
		}
		if sc == sdl.SCANCODE_BACKSPACE {
			if len(c.Input) > 0 {
				c.Input = c.Input[:len(c.Input)-1]
			}
			return
		}
		if sc == sdl.SCANCODE_UP {
			if c.ScrollPos < len(c.Messages) {
				c.ScrollPos++
			}
			return
		}
		if sc == sdl.SCANCODE_DOWN {
			if c.ScrollPos > 0 {
				c.ScrollPos--
			}
			return
		}
		if sc == sdl.SCANCODE_PAGEUP {
			c.ScrollPos += c.MaxVisible
			if c.ScrollPos > len(c.Messages) {
				c.ScrollPos = len(c.Messages)
			}
			return
		}
		if sc == sdl.SCANCODE_PAGEDOWN {
			c.ScrollPos -= c.MaxVisible
			if c.ScrollPos < 0 {
				c.ScrollPos = 0
			}
			return
		}

		shift := kev.Mod&sdl.KMOD_SHIFT != 0
		if ch := scancodeToChar(kev.Scancode, shift); ch != 0 {
			c.Input += string(ch)
		}
	}

	if event.Type == sdl.EVENT_MOUSE_WHEEL {
		wev := event.MouseWheelEvent()
		if wev.Y > 0 {
			c.ScrollPos++
			if c.ScrollPos > len(c.Messages) {
				c.ScrollPos = len(c.Messages)
			}
		} else if wev.Y < 0 {
			c.ScrollPos--
			if c.ScrollPos < 0 {
				c.ScrollPos = 0
			}
		}
	}
}

func (c *Console) Render(renderer *sdl.Renderer, screenW, screenH int32) {
	if !c.Visible {
		return
	}
	renderer.SetDrawColor(0, 0, 0, 180)
	boxH := float32((c.MaxVisible + 2) * 18)
	renderer.RenderFillRect(&sdl.FRect{X: 0, Y: 0, W: float32(screenW), H: boxH})

	total := len(c.Messages)
	start := total - c.MaxVisible - c.ScrollPos
	if start < 0 {
		start = 0
	}
	end := total - c.ScrollPos
	if end > total {
		end = total
	}
	if end < 0 {
		end = 0
	}

	renderer.SetDrawColor(200, 200, 200, 255)
	lineY := float32(4)
	for i := start; i < end; i++ {
		renderer.DebugText(8, lineY, c.Messages[i])
		lineY += 16
	}

	sepY := lineY + 2
	renderer.SetDrawColor(100, 100, 100, 255)
	renderer.RenderFillRect(&sdl.FRect{X: 0, Y: sepY, W: float32(screenW), H: 2})
	renderer.SetDrawColor(0, 255, 0, 255)
	renderer.DebugText(8, sepY+4, "> "+c.Input+"_")
}

func scancodeToChar(sc sdl.Scancode, shift bool) byte {
	lower := map[sdl.Scancode]byte{
		sdl.SCANCODE_A: 'a', sdl.SCANCODE_B: 'b', sdl.SCANCODE_C: 'c',
		sdl.SCANCODE_D: 'd', sdl.SCANCODE_E: 'e', sdl.SCANCODE_F: 'f',
		sdl.SCANCODE_G: 'g', sdl.SCANCODE_H: 'h', sdl.SCANCODE_I: 'i',
		sdl.SCANCODE_J: 'j', sdl.SCANCODE_K: 'k', sdl.SCANCODE_L: 'l',
		sdl.SCANCODE_M: 'm', sdl.SCANCODE_N: 'n', sdl.SCANCODE_O: 'o',
		sdl.SCANCODE_P: 'p', sdl.SCANCODE_Q: 'q', sdl.SCANCODE_R: 'r',
		sdl.SCANCODE_S: 's', sdl.SCANCODE_T: 't', sdl.SCANCODE_U: 'u',
		sdl.SCANCODE_V: 'v', sdl.SCANCODE_W: 'w', sdl.SCANCODE_X: 'x',
		sdl.SCANCODE_Y: 'y', sdl.SCANCODE_Z: 'z',
		sdl.SCANCODE_1: '1', sdl.SCANCODE_2: '2', sdl.SCANCODE_3: '3',
		sdl.SCANCODE_4: '4', sdl.SCANCODE_5: '5', sdl.SCANCODE_6: '6',
		sdl.SCANCODE_7: '7', sdl.SCANCODE_8: '8', sdl.SCANCODE_9: '9',
		sdl.SCANCODE_0: '0',
		sdl.SCANCODE_SPACE:        ' ',
		sdl.SCANCODE_MINUS:        '-',
		sdl.SCANCODE_EQUALS:       '=',
		sdl.SCANCODE_SLASH:        '/',
		sdl.SCANCODE_BACKSLASH:    '\\',
		sdl.SCANCODE_SEMICOLON:    ';',
		sdl.SCANCODE_APOSTROPHE:   '\'',
		sdl.SCANCODE_COMMA:        ',',
		sdl.SCANCODE_PERIOD:       '.',
		sdl.SCANCODE_LEFTBRACKET:  '[',
		sdl.SCANCODE_RIGHTBRACKET: ']',
	}
	upper := map[sdl.Scancode]byte{
		sdl.SCANCODE_1: '!', sdl.SCANCODE_2: '@', sdl.SCANCODE_3: '#',
		sdl.SCANCODE_4: '$', sdl.SCANCODE_5: '%', sdl.SCANCODE_6: '^',
		sdl.SCANCODE_7: '&', sdl.SCANCODE_8: '*', sdl.SCANCODE_9: '(',
		sdl.SCANCODE_0: ')',
		sdl.SCANCODE_MINUS:        '_',
		sdl.SCANCODE_EQUALS:       '+',
		sdl.SCANCODE_SLASH:        '?',
		sdl.SCANCODE_PERIOD:       '>',
		sdl.SCANCODE_COMMA:        '<',
		sdl.SCANCODE_SEMICOLON:    ':',
		sdl.SCANCODE_APOSTROPHE:   '"',
		sdl.SCANCODE_LEFTBRACKET:  '{',
		sdl.SCANCODE_RIGHTBRACKET: '}',
		sdl.SCANCODE_BACKSLASH:    '|',
	}
	if shift {
		if ch, ok := upper[sc]; ok {
			return ch
		}
		if ch, ok := lower[sc]; ok {
			return ch - 32
		}
	}
	if ch, ok := lower[sc]; ok {
		return ch
	}
	return 0
}
