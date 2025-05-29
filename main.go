package main

import (
	. "gowser/url"
	// . "modernc.org/tk9.0"
	"fmt"
	"os"
)

func main() {
	url := NewURL(os.Args[1])
	load(url)

	// canvas := Canvas(Width(800), Height(600))
	// canvas.CreateRectangle(10, 10, 100, 100)
	// canvas.CreateText(50, 50, Txt("Hello, Tk9.0!"))
	// Pack(canvas)
	// App.Center().Wait()
}

func load(url *URL) {
	body := url.Request()
	show(body)
}

func show(body string) {
	inTag := false
	for _, char := range body {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			fmt.Print(string(char))
		}
	}
}
