package browser

import (
	"fmt"
	"gowser/layout"
	"gowser/url"

	tk9_0 "modernc.org/tk9.0"
)

type Chrome struct {
	browser       *Browser
	font          *tk9_0.FontFace
	font_height   float32
	padding       float32
	tabbar_top    float32
	tabbar_bottom float32
	newtab_rect   *layout.Rect
	bottom        float32
	urlbar_top    float32
	urlbar_bottom float32
	back_rect     *layout.Rect
	address_rect  *layout.Rect
	focus         string
	address_bar   string
}

func NewChrome(browser *Browser) *Chrome {
	chrome := &Chrome{browser: browser, address_bar: ""}
	chrome.font = layout.GetFont(20, "normal", "roman")
	chrome.font_height = float32(chrome.font.MetricsLinespace(tk9_0.App))

	chrome.padding = 5
	chrome.tabbar_top = 0
	chrome.tabbar_bottom = chrome.font_height + 2*chrome.padding
	plus_width := layout.Measure(chrome.font, "+") + 2*chrome.padding
	chrome.newtab_rect = layout.NewRect(chrome.padding, chrome.padding, chrome.padding+plus_width, chrome.padding+chrome.font_height)

	chrome.urlbar_top = chrome.tabbar_bottom
	chrome.urlbar_bottom = chrome.urlbar_top + chrome.font_height + 2*chrome.padding

	back_width := layout.Measure(chrome.font, "<") + 2*chrome.padding
	chrome.back_rect = layout.NewRect(
		chrome.padding,
		chrome.urlbar_top+chrome.padding,
		chrome.padding+back_width,
		chrome.urlbar_bottom-chrome.padding,
	)
	chrome.address_rect = layout.NewRect(
		chrome.back_rect.Top+chrome.padding,
		chrome.urlbar_top+chrome.padding,
		DefaultWidth-chrome.padding,
		chrome.urlbar_bottom-chrome.padding,
	)

	chrome.bottom = chrome.urlbar_bottom
	return chrome
}

func (c *Chrome) tab_rect(i int) *layout.Rect {
	tabs_start := c.newtab_rect.Right + c.padding
	tab_width := layout.Measure(c.font, "Tab X") + 2*c.padding
	return layout.NewRect(
		tabs_start+tab_width*float32(i), c.tabbar_top,
		tabs_start+tab_width*float32(i+1), c.tabbar_bottom,
	)
}

func (c *Chrome) paint() []layout.Command {
	cmds := make([]layout.Command, 0)

	cmds = append(cmds, layout.NewDrawRect(layout.NewRect(0, 0, DefaultWidth, c.bottom), "white"))
	cmds = append(cmds, layout.NewDrawLine(0, c.bottom, DefaultWidth, c.bottom, "black", 1))

	cmds = append(cmds, layout.NewDrawOutline(c.newtab_rect, "black", 1))
	cmds = append(cmds, layout.NewDrawText(
		c.newtab_rect.Left+c.padding,
		c.newtab_rect.Top,
		"+", c.font, "black",
	))

	for i, tab := range c.browser.tabs {
		bounds := c.tab_rect(i)
		cmds = append(cmds, layout.NewDrawLine(bounds.Left, 0, bounds.Left, bounds.Bottom, "black", 1))
		cmds = append(cmds, layout.NewDrawLine(bounds.Right, 0, bounds.Right, bounds.Bottom, "black", 1))
		cmds = append(cmds, layout.NewDrawText(bounds.Left+c.padding, bounds.Top+c.padding, fmt.Sprintf("Tab %v", i), c.font, "black"))
		if tab == c.browser.active_tab {
			cmds = append(cmds, layout.NewDrawLine(0, bounds.Bottom, bounds.Left, bounds.Bottom, "black", 1))
			cmds = append(cmds, layout.NewDrawLine(bounds.Right, bounds.Bottom, DefaultWidth, bounds.Bottom, "black", 1))
		}
	}

	cmds = append(cmds, layout.NewDrawOutline(c.back_rect, "black", 1))
	cmds = append(cmds, layout.NewDrawText(
		c.back_rect.Left+c.padding,
		c.back_rect.Top,
		"<", c.font, "black",
	))

	cmds = append(cmds, layout.NewDrawOutline(c.address_rect, "black", 1))
	if c.focus == "address bar" {
		cmds = append(cmds, layout.NewDrawText(
			c.address_rect.Left+c.padding,
			c.address_rect.Top,
			c.address_bar, c.font, "black",
		))
		w := layout.Measure(c.font, c.address_bar)
		cmds = append(cmds, layout.NewDrawLine(
			c.address_rect.Left + c.padding + w,
			c.address_rect.Top,
			c.address_rect.Left + c.padding + w,
			c.address_rect.Bottom,
			"red", 1,
		))
	} else {
		url := c.browser.active_tab.url.String()
		cmds = append(cmds, layout.NewDrawText(
			c.address_rect.Left+c.padding,
			c.address_rect.Top,
			url, c.font, "black",
		))
	}

	return cmds
}

func (c *Chrome) click(x, y float32) {
	c.focus = ""
	if c.newtab_rect.ContainsPoint(x, y) {
		c.browser.NewTab(url.NewURL("https://browser.engineering/"))
	} else if c.back_rect.ContainsPoint(x, y) {
		c.browser.active_tab.go_back()
	} else if c.address_rect.ContainsPoint(x, y) {
		c.focus = "address bar"
		c.address_bar = ""
	} else {
		for i, tab := range c.browser.tabs {
			if c.tab_rect(i).ContainsPoint(x, y) {
				c.browser.active_tab = tab
			}
		}
	}
}

func (c *Chrome) keypress(char rune) {
	if c.focus == "address bar" {
		c.address_bar += string(char)
	}
}

func (c *Chrome) enter() {
	if c.focus == "address bar" {
		c.browser.active_tab.Load(url.NewURL(c.address_bar))
		c.focus = ""
	}
}
