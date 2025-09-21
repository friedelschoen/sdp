package main

import (
	"os"

	"github.com/friedelschoen/slab"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	filename := "example.slab"
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	pres, err := slab.ParsePresentation(file)
	if err != nil {
		panic(err)
	}

	sdl.Init(sdl.INIT_VIDEO)
	defer sdl.Quit()

	win, err := sdl.CreateWindow("slab - "+filename, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	fullscreen := false

	preswin, err := sdl.CreateWindow("slab - Presenter - "+filename, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 1000, 600, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}

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
				preswin.Destroy()
				running = false
			case sdl.WINDOWEVENT_RESIZED:
				fallthrough
			case sdl.WINDOWEVENT_EXPOSED, sdl.WINDOWEVENT_SIZE_CHANGED:
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
				preswin.Destroy()
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

			img, err = preswin.GetSurface()
			if err != nil {
				panic(err)
			}
			slab.DrawPresenter(img, img.Bounds(), pres, index)
			preswin.UpdateSurface()
			dirty = false
		}
	}
}
