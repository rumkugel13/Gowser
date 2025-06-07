package main

import (
	"gowser/browser"
	u "gowser/url"
	"os"

	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	err := sdl.Init(sdl.INIT_EVENTS)
	if err != nil {
		panic("Could not init sdl")
	}

	var url *u.URL
	if len(os.Args) > 1 {
		url = u.NewURL(os.Args[1])
	} else {
		url = u.NewURL("https://browser.engineering/")
	}
	browser := browser.NewBrowser()
	browser.NewTab(url)
	mainloop(browser)
}

func mainloop(browser *browser.Browser) {
	for {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				browser.HandleQuit()
				sdl.Quit()
				os.Exit(0)
			case *sdl.MouseButtonEvent:
				if e.State == sdl.RELEASED {
					continue
				}
				browser.HandleClick(e)
			case *sdl.KeyboardEvent:
				if e.State == sdl.RELEASED {
					continue
				}
				if e.Keysym.Sym == sdl.K_RETURN {
					browser.HandleEnter()
				} else if e.Keysym.Sym == sdl.K_DOWN {
					browser.HandleDown()
				}
			case *sdl.TextInputEvent:
				browser.HandleKey(e)
			}
		}

		sdl.Delay(1)
	}
}
