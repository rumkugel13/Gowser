package browser

import (
	"fmt"
	fnt "gowser/font"
	"gowser/rect"
	"gowser/task"
	"gowser/url"

	"golang.org/x/image/font"
)

type Chrome struct {
	browser       *Browser
	font          font.Face
	font_height   float64
	padding       float64
	tabbar_top    float64
	tabbar_bottom float64
	newtab_rect   *rect.Rect
	bottom        float64
	urlbar_top    float64
	urlbar_bottom float64
	back_rect     *rect.Rect
	address_rect  *rect.Rect
	focus         string
	address_bar   string
}

func NewChrome(browser *Browser) *Chrome {
	chrome := &Chrome{browser: browser, address_bar: ""}
	chrome.font = fnt.GetFont(20, "normal", "roman")
	chrome.font_height = fnt.Linespace(chrome.font)

	chrome.padding = 5
	chrome.tabbar_top = 0
	chrome.tabbar_bottom = chrome.font_height + 2*chrome.padding
	plus_width := fnt.Measure(chrome.font, "+") + 2*chrome.padding
	chrome.newtab_rect = rect.NewRect(chrome.padding, chrome.padding, chrome.padding+plus_width, chrome.padding+chrome.font_height)

	chrome.urlbar_top = chrome.tabbar_bottom
	chrome.urlbar_bottom = chrome.urlbar_top + chrome.font_height + 2*chrome.padding

	back_width := fnt.Measure(chrome.font, "<") + 2*chrome.padding
	chrome.back_rect = rect.NewRect(
		chrome.padding,
		chrome.urlbar_top+chrome.padding,
		chrome.padding+back_width,
		chrome.urlbar_bottom-chrome.padding,
	)
	chrome.address_rect = rect.NewRect(
		chrome.back_rect.Top+chrome.padding,
		chrome.urlbar_top+chrome.padding,
		WIDTH-chrome.padding,
		chrome.urlbar_bottom-chrome.padding,
	)

	chrome.bottom = chrome.urlbar_bottom
	return chrome
}

func (c *Chrome) tab_rect(i int) *rect.Rect {
	tabs_start := c.newtab_rect.Right + c.padding
	tab_width := fnt.Measure(c.font, "Tab X") + 2*c.padding
	return rect.NewRect(
		tabs_start+tab_width*float64(i), c.tabbar_top,
		tabs_start+tab_width*float64(i+1), c.tabbar_bottom,
	)
}

func (c *Chrome) paint() []Command {
	color := "black"
	if c.browser.dark_mode {
		color = "white"
	}
	cmds := make([]Command, 0)

	cmds = append(cmds, NewDrawLine(0, c.bottom, WIDTH, c.bottom, color, 1))

	cmds = append(cmds, NewDrawOutline(c.newtab_rect, color, 1))
	cmds = append(cmds, NewDrawText(
		c.newtab_rect.Left+c.padding,
		c.newtab_rect.Top,
		"+", c.font, color,
	))

	for i, tab := range c.browser.tabs {
		bounds := c.tab_rect(i)
		cmds = append(cmds, NewDrawLine(bounds.Left, 0, bounds.Left, bounds.Bottom, color, 1))
		cmds = append(cmds, NewDrawLine(bounds.Right, 0, bounds.Right, bounds.Bottom, color, 1))
		cmds = append(cmds, NewDrawText(bounds.Left+c.padding, bounds.Top+c.padding, fmt.Sprintf("Tab %v", i), c.font, color))
		if tab == c.browser.ActiveTab {
			cmds = append(cmds, NewDrawLine(0, bounds.Bottom, bounds.Left, bounds.Bottom, color, 1))
			cmds = append(cmds, NewDrawLine(bounds.Right, bounds.Bottom, WIDTH, bounds.Bottom, color, 1))
		}
	}

	cmds = append(cmds, NewDrawOutline(c.back_rect, color, 1))
	cmds = append(cmds, NewDrawText(
		c.back_rect.Left+c.padding,
		c.back_rect.Top,
		"<", c.font, color,
	))

	cmds = append(cmds, NewDrawOutline(c.address_rect, color, 1))
	if c.focus == "address bar" {
		cmds = append(cmds, NewDrawText(
			c.address_rect.Left+c.padding,
			c.address_rect.Top,
			c.address_bar, c.font, color,
		))
		w := fnt.Measure(c.font, c.address_bar)
		cmds = append(cmds, NewDrawLine(
			c.address_rect.Left+c.padding+w,
			c.address_rect.Top,
			c.address_rect.Left+c.padding+w,
			c.address_rect.Bottom,
			"red", 1,
		))
	} else {
		var url string
		if c.browser.active_tab_url != nil {
			url = c.browser.active_tab_url.String()
		}
		cmds = append(cmds, NewDrawText(
			c.address_rect.Left+c.padding,
			c.address_rect.Top,
			url, c.font, color,
		))
	}

	return cmds
}

func (c *Chrome) click(x, y float64) {
	c.focus = ""
	if c.newtab_rect.ContainsPoint(x, y) {
		new_url, err := url.NewURL("https://browser.engineering/")
		if err != nil {
			fmt.Println("Creating URL failed: " + err.Error())
			return
		}
		c.browser.new_tab_internal(new_url)
	} else if c.back_rect.ContainsPoint(x, y) {
		task := task.NewTask(func(i ...interface{}) {
			c.browser.ActiveTab.go_back()
		})
		c.browser.ActiveTab.TaskRunner.ScheduleTask(task)
	} else if c.address_rect.ContainsPoint(x, y) {
		c.focus = "address bar"
		c.address_bar = ""
	} else {
		for i, tab := range c.browser.tabs {
			if c.tab_rect(i).ContainsPoint(x, y) {
				c.browser.set_active_tab(tab)
				active_tab := c.browser.ActiveTab
				task := task.NewTask(func(i ...interface{}) {
					active_tab.SetNeedsRenderAllFrames()
				})
				active_tab.TaskRunner.ScheduleTask(task)
				break
			}
		}
	}
}

func (c *Chrome) keypress(char rune) bool {
	if c.focus == "address bar" {
		c.address_bar += string(char)
		return true
	}
	return false
}

func (c *Chrome) backspace() bool {
	if c.focus == "address bar" {
		if len(c.address_bar) > 0 {
			c.address_bar = c.address_bar[:len(c.address_bar)-1]
		}
		return true
	}
	return false
}

func (c *Chrome) enter() bool {
	if c.focus == "address bar" {
		new_url, err := url.NewURL(c.address_bar)
		if err != nil {
			fmt.Println("Creating URL failed: " + err.Error())
			return false
		}
		c.browser.ActiveTab.browser.ScheduleLoad(new_url, "")
		c.focus = ""
		return true
	}
	return false
}

func (c *Chrome) blur() {
	c.focus = ""
}

func (c *Chrome) focus_addressbar() {
	c.focus = "address bar"
	c.address_bar = ""
}
