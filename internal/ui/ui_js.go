// Copyright 2015 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build js

package ui

import (
	"strconv"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

func Now() int64 {
	// time.Now() is not reliable until GopherJS supports performance.now().
	return int64(js.Global.Get("performance").Call("now").Float() * float64(time.Millisecond))
}

func (u *UserInterface) SetScreenSize(width, height int) bool {
	scale := canvas.Get("dataset").Get("ebitenScreenScale").Int()
	return u.setScreenSize(width, height, scale)
}

func (u *UserInterface) SetScreenScale(scale int) bool {
	width, height := currentUI.size()
	return u.setScreenSize(width, height, scale)
}

func (u *UserInterface) ScreenScale() int {
	return canvas.Get("dataset").Get("ebitenScreenScale").Int()
}

func (u *UserInterface) ActualScreenScale() int {
	return canvas.Get("dataset").Get("ebitenActualScreenScale").Int()
}

var canvas *js.Object

type UserInterface struct{}

var currentUI = &UserInterface{}

func CurrentUI() *UserInterface {
	return currentUI
}

// NOTE: This returns true even when the browser is not active.
func shown() bool {
	return !js.Global.Get("document").Get("hidden").Bool()
}

func vsync() {
	ch := make(chan struct{})
	n := 1
	var l func()
	l = func() {
		if 0 < n {
			n--
			// TODO: In iOS8, this is called at every 1/30[sec] frame.
			// Can we use DOMHighResTimeStamp?
			js.Global.Get("window").Call("requestAnimationFrame", l)
			return
		}
		close(ch)
	}
	l()
	<-ch
}

func (u *UserInterface) DoEvents() error {
	currentInput.UpdateGamepads()
	return nil
}

func (u *UserInterface) Terminate() {
	// Do nothing.
}

func (u *UserInterface) IsClosed() bool {
	return false
}

func (u *UserInterface) SwapBuffers() {
	vsync()
	for !shown() {
		vsync()
	}
}

func Init() *opengl.Context {
	// Do nothing in node.js.
	if js.Global.Get("require") != js.Undefined {
		return opengl.NewContext()
	}

	doc := js.Global.Get("document")
	window := js.Global.Get("window")
	if doc.Get("body") == nil {
		ch := make(chan struct{})
		window.Call("addEventListener", "load", func() {
			close(ch)
		})
		<-ch
	}

	canvas = doc.Call("createElement", "canvas")
	canvas.Set("width", 16)
	canvas.Set("height", 16)
	doc.Get("body").Call("appendChild", canvas)

	htmlStyle := doc.Get("documentElement").Get("style")
	htmlStyle.Set("height", "100%")
	htmlStyle.Set("margin", "0")
	htmlStyle.Set("padding", "0")

	bodyStyle := doc.Get("body").Get("style")
	bodyStyle.Set("backgroundColor", "#000")
	bodyStyle.Set("position", "relative")
	bodyStyle.Set("height", "100%")
	bodyStyle.Set("margin", "0")
	bodyStyle.Set("padding", "0")
	// TODO: This is OK as long as the game is in an independent iframe.
	// What if the canvas is embedded in a HTML directly?
	doc.Get("body").Call("addEventListener", "click", func() {
		canvas.Call("focus")
	})

	canvasStyle := canvas.Get("style")
	canvasStyle.Set("position", "absolute")

	// Make the canvas focusable.
	canvas.Call("setAttribute", "tabindex", 1)
	canvas.Get("style").Set("outline", "none")

	// Keyboard
	canvas.Call("addEventListener", "keydown", func(e *js.Object) {
		e.Call("preventDefault")
		code := e.Get("keyCode").Int()
		currentInput.KeyDown(code)
	})
	canvas.Call("addEventListener", "keyup", func(e *js.Object) {
		e.Call("preventDefault")
		code := e.Get("keyCode").Int()
		currentInput.KeyUp(code)
	})

	// Mouse
	canvas.Call("addEventListener", "mousedown", func(e *js.Object) {
		e.Call("preventDefault")
		button := e.Get("button").Int()
		currentInput.MouseDown(button)
		setMouseCursorFromEvent(e)
	})
	canvas.Call("addEventListener", "mouseup", func(e *js.Object) {
		e.Call("preventDefault")
		button := e.Get("button").Int()
		currentInput.MouseUp(button)
		setMouseCursorFromEvent(e)
	})
	canvas.Call("addEventListener", "mousemove", func(e *js.Object) {
		e.Call("preventDefault")
		setMouseCursorFromEvent(e)
	})
	canvas.Call("addEventListener", "contextmenu", func(e *js.Object) {
		e.Call("preventDefault")
	})

	// Touch (emulating mouse events)
	// TODO: Create indimendent touch functions
	canvas.Call("addEventListener", "touchstart", func(e *js.Object) {
		e.Call("preventDefault")
		currentInput.MouseDown(0)
		touches := e.Get("changedTouches")
		touch := touches.Index(0)
		setMouseCursorFromEvent(touch)
	})
	canvas.Call("addEventListener", "touchend", func(e *js.Object) {
		e.Call("preventDefault")
		currentInput.MouseUp(0)
		touches := e.Get("changedTouches")
		touch := touches.Index(0)
		setMouseCursorFromEvent(touch)
	})
	canvas.Call("addEventListener", "touchmove", func(e *js.Object) {
		e.Call("preventDefault")
		touches := e.Get("changedTouches")
		touch := touches.Index(0)
		setMouseCursorFromEvent(touch)
	})

	// Gamepad
	window.Call("addEventListener", "gamepadconnected", func(e *js.Object) {
		// Do nothing.
	})

	return opengl.NewContext()
}

func setMouseCursorFromEvent(e *js.Object) {
	scale := canvas.Get("dataset").Get("ebitenScreenScale").Int()
	rect := canvas.Call("getBoundingClientRect")
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	x -= rect.Get("left").Int()
	y -= rect.Get("top").Int()
	currentInput.SetMouseCursor(x/scale, y/scale)
}

func devicePixelRatio() int {
	// TODO: What if ratio is not an integer but a float?
	ratio := js.Global.Get("window").Get("devicePixelRatio").Int()
	if ratio == 0 {
		ratio = 1
	}
	return ratio
}

func (u *UserInterface) Start(width, height, scale int, title string) error {
	doc := js.Global.Get("document")
	doc.Set("title", title)
	u.setScreenSize(width, height, scale)
	canvas.Call("focus")
	return nil
}

func (*UserInterface) size() (width, height int) {
	a := canvas.Get("dataset").Get("ebitenActualScreenScale").Int()
	if a == 0 {
		// a == 0 only on the initial state.
		return
	}
	width = canvas.Get("width").Int() / a
	height = canvas.Get("height").Int() / a
	return
}

func (u *UserInterface) setScreenSize(width, height, scale int) bool {
	w, h := u.size()
	s := canvas.Get("dataset").Get("ebitenScreenScale").Int()
	if w == width && h == height && s == scale {
		return false
	}

	actualScreenScale := scale * devicePixelRatio()
	canvas.Set("width", width*actualScreenScale)
	canvas.Set("height", height*actualScreenScale)
	canvas.Get("dataset").Set("ebitenScreenScale", scale)
	canvas.Get("dataset").Set("ebitenActualScreenScale", actualScreenScale)
	canvasStyle := canvas.Get("style")

	cssWidth := width * scale
	cssHeight := height * scale
	canvasStyle.Set("width", strconv.Itoa(cssWidth)+"px")
	canvasStyle.Set("height", strconv.Itoa(cssHeight)+"px")
	// CSS calc requires space chars.
	canvasStyle.Set("left", "calc((100% - "+strconv.Itoa(cssWidth)+"px) / 2)")
	canvasStyle.Set("top", "calc((100% - "+strconv.Itoa(cssHeight)+"px) / 2)")
	return true
}
