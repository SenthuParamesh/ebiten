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

package js

import (
	"encoding/hex"
	"syscall/js"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/jsutil"
)

var (
	stringKeydown    = js.ValueOf("keydown")
	stringKeypress   = js.ValueOf("keypress")
	stringKeyup      = js.ValueOf("keyup")
	stringMousedown  = js.ValueOf("mousedown")
	stringMouseup    = js.ValueOf("mouseup")
	stringMousemove  = js.ValueOf("mousemove")
	stringWheel      = js.ValueOf("wheel")
	stringTouchstart = js.ValueOf("touchstart")
	stringTouchend   = js.ValueOf("touchend")
	stringTouchmove  = js.ValueOf("touchmove")
)

var jsKeys []js.Value

func init() {
	for _, k := range driverKeyToJSKey {
		jsKeys = append(jsKeys, k)
	}
}

func jsKeyToID(key js.Value) int {
	// js.Value cannot be used as a map key.
	// As the number of keys is around 100, just a dumb loop should work.
	for i, k := range jsKeys {
		if jsutil.Equal(k, key) {
			return i
		}
	}
	return -1
}

type pos struct {
	X int
	Y int
}

type gamePad struct {
	valid         bool
	name          string
	axisNum       int
	axes          [16]float64
	buttonNum     int
	buttonPressed [256]bool
}

type Input struct {
	keyPressed         map[int]bool
	keyPressedEdge     map[int]bool
	mouseButtonPressed map[int]bool
	cursorX            int
	cursorY            int
	wheelX             float64
	wheelY             float64
	gamepads           [16]gamePad
	touches            map[driver.TouchID]pos
	runeBuffer         []rune
	ui                 *UserInterface
}

func (i *Input) CursorPosition() (x, y int) {
	xf, yf := i.ui.context.AdjustPosition(float64(i.cursorX), float64(i.cursorY), i.ui.DeviceScaleFactor())
	return int(xf), int(yf)
}

func (i *Input) GamepadSDLID(id driver.GamepadID) string {
	// This emulates the implementation of EMSCRIPTEN_JoystickGetDeviceGUID.
	// https://hg.libsdl.org/SDL/file/bc90ce38f1e2/src/joystick/emscripten/SDL_sysjoystick.c#l385
	if len(i.gamepads) <= int(id) {
		return ""
	}
	var sdlid [16]byte
	copy(sdlid[:], []byte(i.gamepads[id].name))
	return hex.EncodeToString(sdlid[:])
}

// GamepadName returns a string containing some information about the controller.
// A PS2 controller returned "810-3-USB Gamepad" on Firefox
// A Xbox 360 controller returned "xinput" on Firefox and "Xbox 360 Controller (XInput STANDARD GAMEPAD)" on Chrome
func (i *Input) GamepadName(id driver.GamepadID) string {
	if len(i.gamepads) <= int(id) {
		return ""
	}
	return i.gamepads[id].name
}

func (i *Input) GamepadIDs() []driver.GamepadID {
	if len(i.gamepads) == 0 {
		return nil
	}
	var r []driver.GamepadID
	for id, g := range i.gamepads {
		if g.valid {
			r = append(r, driver.GamepadID(id))
		}
	}
	return r
}

func (i *Input) GamepadAxisNum(id driver.GamepadID) int {
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].axisNum
}

func (i *Input) GamepadAxis(id driver.GamepadID, axis int) float64 {
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].axes[axis]
}

func (i *Input) GamepadButtonNum(id driver.GamepadID) int {
	if len(i.gamepads) <= int(id) {
		return 0
	}
	return i.gamepads[id].buttonNum
}

func (i *Input) IsGamepadButtonPressed(id driver.GamepadID, button driver.GamepadButton) bool {
	if len(i.gamepads) <= int(id) {
		return false
	}
	return i.gamepads[id].buttonPressed[button]
}

func (i *Input) TouchIDs() []driver.TouchID {
	if len(i.touches) == 0 {
		return nil
	}

	var ids []driver.TouchID
	for id := range i.touches {
		ids = append(ids, id)
	}
	return ids
}

func (i *Input) TouchPosition(id driver.TouchID) (x, y int) {
	d := i.ui.DeviceScaleFactor()
	for tid, pos := range i.touches {
		if id == tid {
			x, y := i.ui.context.AdjustPosition(float64(pos.X), float64(pos.Y), d)
			return int(x), int(y)
		}
	}
	return 0, 0
}

func (i *Input) RuneBuffer() []rune {
	rs := make([]rune, len(i.runeBuffer))
	copy(rs, i.runeBuffer)
	return rs
}

func (i *Input) resetForFrame() {
	i.runeBuffer = nil
	i.wheelX = 0
	i.wheelY = 0
}

func (i *Input) IsKeyPressed(key driver.Key) bool {
	if i.keyPressed != nil {
		if i.keyPressed[jsKeyToID(driverKeyToJSKey[key])] {
			return true
		}
	}
	if i.keyPressedEdge != nil {
		for c, k := range edgeKeyCodeToDriverKey {
			if k != key {
				continue
			}
			if i.keyPressedEdge[c] {
				return true
			}
		}
	}
	return false
}

var codeToMouseButton = map[int]driver.MouseButton{
	0: driver.MouseButtonLeft,
	1: driver.MouseButtonMiddle,
	2: driver.MouseButtonRight,
}

func (i *Input) IsMouseButtonPressed(button driver.MouseButton) bool {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	for c, b := range codeToMouseButton {
		if b != button {
			continue
		}
		if i.mouseButtonPressed[c] {
			return true
		}
	}
	return false
}

func (i *Input) Wheel() (xoff, yoff float64) {
	return i.wheelX, i.wheelY
}

func (i *Input) keyDown(code js.Value) {
	if i.keyPressed == nil {
		i.keyPressed = map[int]bool{}
	}
	i.keyPressed[jsKeyToID(code)] = true
}

func (i *Input) keyUp(code js.Value) {
	if i.keyPressed == nil {
		i.keyPressed = map[int]bool{}
	}
	i.keyPressed[jsKeyToID(code)] = false
}

func (i *Input) keyDownEdge(code int) {
	if i.keyPressedEdge == nil {
		i.keyPressedEdge = map[int]bool{}
	}
	i.keyPressedEdge[code] = true
}

func (i *Input) keyUpEdge(code int) {
	if i.keyPressedEdge == nil {
		i.keyPressedEdge = map[int]bool{}
	}
	i.keyPressedEdge[code] = false
}

func (i *Input) mouseDown(code int) {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = true
}

func (i *Input) mouseUp(code int) {
	if i.mouseButtonPressed == nil {
		i.mouseButtonPressed = map[int]bool{}
	}
	i.mouseButtonPressed[code] = false
}

func (i *Input) setMouseCursor(x, y int) {
	i.cursorX, i.cursorY = x, y
}

func (i *Input) updateGamepads() {
	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return
	}
	if !nav.Get("getGamepads").Truthy() {
		return
	}
	gamepads := nav.Call("getGamepads")
	l := gamepads.Get("length").Int()
	for id := 0; id < l; id++ {
		i.gamepads[id].valid = false
		gamepad := gamepads.Index(id)
		if !gamepad.Truthy() {
			continue
		}
		i.gamepads[id].valid = true
		i.gamepads[id].name = gamepad.Get("id").String()

		axes := gamepad.Get("axes")
		axesNum := axes.Get("length").Int()
		i.gamepads[id].axisNum = axesNum
		for a := 0; a < len(i.gamepads[id].axes); a++ {
			if axesNum <= a {
				i.gamepads[id].axes[a] = 0
				continue
			}
			i.gamepads[id].axes[a] = axes.Index(a).Float()
		}

		buttons := gamepad.Get("buttons")
		buttonsNum := buttons.Get("length").Int()
		i.gamepads[id].buttonNum = buttonsNum
		for b := 0; b < len(i.gamepads[id].buttonPressed); b++ {
			if buttonsNum <= b {
				i.gamepads[id].buttonPressed[b] = false
				continue
			}
			i.gamepads[id].buttonPressed[b] = buttons.Index(b).Get("pressed").Bool()
		}
	}
}

func (i *Input) updateFromEvent(e js.Value) {
	// Avoid using js.Value.String() as String creates a Uint8Array via a TextEncoder and causes a heavy
	// overhead (#1437).
	switch t := e.Get("type"); {
	case jsutil.Equal(t, stringKeydown):
		c := e.Get("code")
		if c.Type() != js.TypeString {
			code := e.Get("keyCode").Int()
			if edgeKeyCodeToDriverKey[code] == driver.KeyUp ||
				edgeKeyCodeToDriverKey[code] == driver.KeyDown ||
				edgeKeyCodeToDriverKey[code] == driver.KeyLeft ||
				edgeKeyCodeToDriverKey[code] == driver.KeyRight ||
				edgeKeyCodeToDriverKey[code] == driver.KeyBackspace ||
				edgeKeyCodeToDriverKey[code] == driver.KeyTab {
				e.Call("preventDefault")
			}
			i.keyDownEdge(code)
			return
		}
		if jsutil.Equal(c, driverKeyToJSKey[driver.KeyUp]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyDown]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyLeft]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyRight]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyBackspace]) ||
			jsutil.Equal(c, driverKeyToJSKey[driver.KeyTab]) {
			e.Call("preventDefault")
		}
		i.keyDown(c)
	case jsutil.Equal(t, stringKeypress):
		if r := rune(e.Get("charCode").Int()); unicode.IsPrint(r) {
			i.runeBuffer = append(i.runeBuffer, r)
		}
	case jsutil.Equal(t, stringKeyup):
		if e.Get("code").Type() != js.TypeString {
			// Assume that UA is Edge.
			code := e.Get("keyCode").Int()
			i.keyUpEdge(code)
			return
		}
		i.keyUp(e.Get("code"))
	case jsutil.Equal(t, stringMousedown):
		button := e.Get("button").Int()
		i.mouseDown(button)
		i.setMouseCursorFromEvent(e)
	case jsutil.Equal(t, stringMouseup):
		button := e.Get("button").Int()
		i.mouseUp(button)
		i.setMouseCursorFromEvent(e)
	case jsutil.Equal(t, stringMousemove):
		i.setMouseCursorFromEvent(e)
	case jsutil.Equal(t, stringWheel):
		// TODO: What if e.deltaMode is not DOM_DELTA_PIXEL?
		i.wheelX = -e.Get("deltaX").Float()
		i.wheelY = -e.Get("deltaY").Float()
	case jsutil.Equal(t, stringTouchstart) || jsutil.Equal(t, stringTouchend) || jsutil.Equal(t, stringTouchmove):
		i.updateTouchesFromEvent(e)
	}
}

func (i *Input) setMouseCursorFromEvent(e js.Value) {
	x, y := e.Get("clientX").Int(), e.Get("clientY").Int()
	i.setMouseCursor(x, y)
}

func (i *Input) updateTouchesFromEvent(e js.Value) {
	j := e.Get("targetTouches")
	ts := map[driver.TouchID]pos{}
	for i := 0; i < j.Length(); i++ {
		jj := j.Call("item", i)
		id := driver.TouchID(jj.Get("identifier").Int())
		ts[id] = pos{
			X: jj.Get("clientX").Int(),
			Y: jj.Get("clientY").Int(),
		}
	}
	i.touches = ts
}

func (i *Input) updateForGo2Cpp() {
	if !go2cpp.Truthy() {
		return
	}

	i.touches = map[driver.TouchID]pos{}
	maxID := go2cpp.Get("touchCount").Int()
	for idx := 0; idx < maxID; idx++ {
		id := go2cpp.Call("getTouchPositionId", idx)
		x := go2cpp.Call("getTouchPositionX", idx)
		y := go2cpp.Call("getTouchPositionY", idx)
		i.touches[driver.TouchID(id.Int())] = pos{
			X: x.Int(),
			Y: y.Int(),
		}
	}
}
