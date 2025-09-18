package main

import (
	"os"

	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	filename := "example.sdp"
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	pres, err := ParsePresentation(file)
	if err != nil {
		panic(err)
	}

	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()

	win, err := sdl.CreateWindow("Simple Descriptive Presentations - "+filename, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	fullscreen := false

	index := 0
	running := true
	for running {
		ev := sdl.WaitEvent()

		dirty := false
		switch ev := ev.(type) {
		case *sdl.QuitEvent:
			running = false
		case *sdl.WindowEvent:
			switch ev.Event {
			case sdl.WINDOWEVENT_CLOSE:
				win.Destroy()
				running = false
			case sdl.WINDOWEVENT_EXPOSED, sdl.WINDOWEVENT_RESIZED, sdl.WINDOWEVENT_SIZE_CHANGED:
				dirty = true
			}
		case *sdl.KeyboardEvent:
			if ev.Type != sdl.KEYDOWN {
				break
			}
			switch ev.Keysym.Sym {
			case sdl.K_UP, sdl.K_LEFT:
				if index > 0 {
					index--
					dirty = true
				}
			case sdl.K_DOWN, sdl.K_RIGHT:
				if index < len(pres.Slides)-1 {
					index++
					dirty = true
				}

			case sdl.K_f:
				if fullscreen {
					win.SetFullscreen(0)
					fullscreen = false
				} else {
					win.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
					fullscreen = true
				}
			case sdl.K_q:
				win.Destroy()
				running = false
			}
		}
		if running == false {
			break
		}

		if dirty {
			img, err := win.GetSurface()
			if err != nil {
				panic(err)
			}
			pres.Slides[index].Draw(img, img.Bounds())
			win.UpdateSurface()
			dirty = false
		}
	}
}
