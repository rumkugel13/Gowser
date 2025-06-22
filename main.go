package main

import (
	"fmt"
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

	url_str := "https://browser.engineering/"
	if len(os.Args) > 1 {
		url_str = os.Args[1]
	}
	browser := browser.NewBrowser()
	url, err := u.NewURL(url_str)
	if err != nil {
		panic("Invalid url: " + err.Error())
	}
	browser.NewTab(url)
	browser.CompositeRasterAndDraw()
	mainloop(browser)
}

func mainloop(browser *browser.Browser) {
	ctrl_down := false
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
			case *sdl.MouseMotionEvent:
				browser.HandleHover(float64(e.X), float64(e.Y))
			case *sdl.MouseWheelEvent:
				if e.Y < 0 {
					browser.HandleDown()
				} else {
					browser.HandleUp()
				}
			case *sdl.KeyboardEvent:
				if e.State == sdl.RELEASED {
					if e.Keysym.Sym == sdl.K_RCTRL || e.Keysym.Sym == sdl.K_LCTRL {
						ctrl_down = false
					}
				} else if e.State == sdl.PRESSED {
					if ctrl_down {
						if e.Keysym.Sym == sdl.K_PLUS {
							browser.IncrementZoom(true)
						} else if e.Keysym.Sym == sdl.K_MINUS {
							browser.IncrementZoom(false)
						} else if e.Keysym.Sym == sdl.K_0 {
							browser.ResetZoom()
						} else if e.Keysym.Sym == sdl.K_d {
							browser.ToggleDarkMode()
						} else if e.Keysym.Sym == sdl.K_LEFT {
							browser.GoBack()
						} else if e.Keysym.Sym == sdl.K_l {
							browser.FocusAddressbar()
						} else if e.Keysym.Sym == sdl.K_t {
							new_url, err := u.NewURL("https://browser.engineering/")
							if err != nil {
								fmt.Println("Invalid URL:", err.Error())
							} else {
								browser.NewTab(new_url)
							}
						} else if e.Keysym.Sym == sdl.K_TAB {
							browser.CycleTabs()
						} else if e.Keysym.Sym == sdl.K_q {
							browser.HandleQuit()
							sdl.Quit()
							os.Exit(0)
							break
						} else if e.Keysym.Sym == sdl.K_a {
							browser.ToggleAccessibility()
						}
					} else {
						if e.Keysym.Sym == sdl.K_RCTRL || e.Keysym.Sym == sdl.K_LCTRL {
							ctrl_down = true
						} else if e.Keysym.Sym == sdl.K_RETURN {
							browser.HandleEnter()
						} else if e.Keysym.Sym == sdl.K_TAB {
							browser.HandleTab()
						} else if e.Keysym.Sym == sdl.K_BACKSPACE {
							browser.HandleBackspace()
						} else if e.Keysym.Sym == sdl.K_UP {
							browser.HandleUp()
						} else if e.Keysym.Sym == sdl.K_DOWN {
							browser.HandleDown()
						}
					}
				}
			case *sdl.TextInputEvent:
				browser.HandleKey(e)
			}
		}
		browser.CompositeRasterAndDraw()
		browser.ScheduleAnimationFrame()

		sdl.Delay(1)
	}
}
