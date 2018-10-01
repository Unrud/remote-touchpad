// +build x11

/*
 *    Copyright (c) 2018 Unrud<unrud@outlook.com>
 *
 *    This file is part of Remote-Touchpad.
 *
 *    Foobar is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU General Public License as published by
 *    the Free Software Foundation, either version 3 of the License, or
 *    (at your option) any later version.
 *
 *    Remote-Touchpad is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with Foobar.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

// #cgo LDFLAGS: -lX11 -lXtst
// #include <X11/Xlib.h>
// #include <X11/Intrinsic.h>
// #include <X11/extensions/XTest.h>
// #include <X11/XKBlib.h>
import "C"
import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"
)

const (
	keyboardMappingDelay time.Duration = 125 * time.Millisecond
	scrollDiv            int           = 20
)

var modifierIndices [6]uint = [...]uint{C.ShiftMapIndex, C.Mod1MapIndex,
	C.Mod2MapIndex, C.Mod3MapIndex, C.Mod4MapIndex, C.Mod5MapIndex}

type x11Plugin struct {
	display                          *C.Display
	lock                             sync.Mutex
	scrollHorizontal, scrollVertical int
}

func InitX11Plugin() (Plugin, error) {
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType != "" && sessionType != "x11" {
		return nil, UnsupportedPlatformError{errors.New(fmt.Sprintf(
			"unsupported session type '%v'", sessionType))}
	}
	display := C.XOpenDisplay(nil)
	if display == nil {
		return nil, UnsupportedPlatformError{
			errors.New("failed to connect to X server")}
	}
	return &x11Plugin{display: display}, nil
}

func (p *x11Plugin) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.display == nil {
		return errors.New("X server connection closed")
	}
	C.XCloseDisplay(p.display)
	p.display = nil
	return nil
}

func (p *x11Plugin) findEmptyKeycodeLocked() (C.KeyCode, C.int, error) {
	var minKeycodes, maxKeycodes C.int
	C.XDisplayKeycodes(p.display, &minKeycodes, &maxKeycodes)
	var keysymsPerKeycode C.int
	keysyms := C.XGetKeyboardMapping(p.display, C.KeyCode(minKeycodes),
		maxKeycodes-minKeycodes+1, &keysymsPerKeycode)
	if keysyms == nil {
		return 0, 0, errors.New("failed to get keyboard mapping")
	}
	defer C.XFree(unsafe.Pointer(keysyms))
keycodes:
	for keycode := C.KeyCode(minKeycodes); keycode <= C.KeyCode(maxKeycodes); keycode++ {
		for i := 0; i < int(keysymsPerKeycode); i++ {
			keysymsIndex := int(keycode-
				C.KeyCode(minKeycodes))*int(keysymsPerKeycode) + i
			keysym := *(*C.KeySym)(unsafe.Pointer(uintptr(unsafe.Pointer(keysyms)) +
				uintptr(keysymsIndex)*unsafe.Sizeof(*keysyms)))
			if keysym != 0 {
				continue keycodes
			}
		}
		return keycode, keysymsPerKeycode, nil
	}
	return 0, 0, errors.New("no empty keycode found")
}

func (p *x11Plugin) changeKeyMappingLocked(keysymsPerKeycode C.int,
	keycode C.KeyCode, keysym Keysym) {
	keycodeMapping := make([]C.KeySym, keysymsPerKeycode)
	for i := range keycodeMapping {
		keycodeMapping[i] = C.KeySym(keysym)
	}
	C.XChangeKeyboardMapping(p.display, C.int(keycode), keysymsPerKeycode,
		(*C.KeySym)(unsafe.Pointer(&keycodeMapping[0])), 1)
	C.XFlush(p.display)
}

func (p *x11Plugin) getModKeycodesLocked() map[uint]C.KeyCode {
	modKeymap := C.XGetModifierMapping(p.display)
	defer C.XFreeModifiermap(modKeymap)
	modKeycodes := make(map[uint]C.KeyCode)
	for _, modIndex := range modifierIndices {
		for i := 0; i < int(modKeymap.max_keypermod); i++ {
			keycode := *(*C.KeyCode)(unsafe.Pointer(uintptr(unsafe.Pointer(modKeymap.modifiermap)) +
				uintptr(uint(modIndex)*uint(modKeymap.max_keypermod)+uint(i))))
			if keycode != 0 {
				modKeycodes[1<<uint(modIndex)] = keycode
				break
			}
		}
	}
	return modKeycodes
}

func (p *x11Plugin) findKeycodeLocked(keyboard C.XkbDescPtr,
	modKeycodes map[uint]C.KeyCode, activeMods C.uint,
	keysym Keysym) (C.KeyCode, C.uint) {
	keycode := C.XKeysymToKeycode(p.display, C.KeySym(keysym))
	if keycode == 0 {
		return 0, 0
	}
	var alwaysActiveMods C.uint
	for modIndex := uint(0); modIndex < 8; modIndex++ {
		mod := uint(1) << modIndex
		if _, modAvailable := modKeycodes[mod]; !modAvailable {
			alwaysActiveMods |= activeMods & C.uint(mod)
		}
	}
	_, shiftModAvailable := modKeycodes[C.ShiftMask]
	for _, modIndex := range modifierIndices {
		var mod C.uint
		if modIndex != C.ShiftMapIndex {
			mod = 1 << modIndex
		}
		for _, shiftMod := range [...]C.uint{0, C.ShiftMask} {
			if shiftMod != 0 && !shiftModAvailable {
				continue
			}
			mods := alwaysActiveMods | shiftMod | mod
			var retMods C.uint
			var retKeysym C.KeySym
			C.XkbTranslateKeyCode(keyboard, keycode, mods, &retMods, &retKeysym)
			if retKeysym == C.KeySym(keysym) {
				return keycode, mods
			}
		}
	}
	return 0, 0
}

func (p *x11Plugin) sendModsLocked(modKeycodes map[uint]C.KeyCode, mods C.uint,
	press bool) {
	var pressC C.int = C.False
	if press {
		pressC = C.True
	}
	for mod, keycode := range modKeycodes {
		if mods&C.uint(mod) != 0 {
			C.XTestFakeKeyEvent(p.display, C.uint(keycode), pressC, 0)
		}
	}
}

func (p *x11Plugin) keyboardKeys(keys []Keysym) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.display == nil {
		return errors.New("X server connection closed")
	}
	if len(keys) == 0 {
		return nil
	}
	rootWindow := C.XDefaultRootWindow(p.display)
	modKeycodes := p.getModKeycodesLocked()
	keyboard := C.XkbGetKeyboard(p.display,
		C.XkbCompatMapMask|C.XkbGeometryMask, C.XkbUseCoreKbd)
	defer C.XkbFreeKeyboard(keyboard, C.XkbAllComponentsMask, C.True)
	var emptyKeycode C.KeyCode
	var keysymsPerKeycode C.int
	for _, keysym := range keys {
		var root, child C.Window
		var rootX, rootY, x, y C.int
		var activeMods C.uint
		C.XSync(p.display, C.False)
		C.XQueryPointer(p.display, rootWindow, &root, &child, &rootX, &rootY,
			&x, &y, &activeMods)
		keycode, mods := p.findKeycodeLocked(keyboard, modKeycodes, activeMods,
			keysym)
		var pressMods, releaseMods C.uint
		if keycode == 0 {
			if emptyKeycode == 0 {
				var err error
				emptyKeycode, keysymsPerKeycode, err = p.findEmptyKeycodeLocked()
				if err != nil {
					return err
				}
				defer p.changeKeyMappingLocked(keysymsPerKeycode, emptyKeycode, 0)
			}
			keycode = emptyKeycode
			p.changeKeyMappingLocked(keysymsPerKeycode, keycode, keysym)
			// race condition!
			time.Sleep(keyboardMappingDelay)
		} else {
			pressMods = mods & ^activeMods
			releaseMods = activeMods & ^mods
		}
		p.sendModsLocked(modKeycodes, releaseMods, false)
		p.sendModsLocked(modKeycodes, pressMods, true)
		C.XTestFakeKeyEvent(p.display, C.uint(keycode), C.True, 0)
		C.XTestFakeKeyEvent(p.display, C.uint(keycode), C.False, 0)
		p.sendModsLocked(modKeycodes, pressMods, false)
		p.sendModsLocked(modKeycodes, releaseMods, true)
		C.XFlush(p.display)
		if keycode == emptyKeycode {
			// race condition!
			time.Sleep(keyboardMappingDelay)
		}
	}
	return nil
}

func (p *x11Plugin) KeyboardText(text string) error {
	keys := make([]Keysym, 0, len(text))
	for _, runeValue := range text {
		keysym, err := RuneToKeysym(runeValue)
		if err != nil {
			return err
		}
		keys = append(keys, keysym)
	}
	return p.keyboardKeys(keys)
}

func (p *x11Plugin) KeyboardKey(key Key) error {
	keysym, err := KeyToKeysym(key)
	if err != nil {
		return err
	}
	keys := [...]Keysym{keysym}
	return p.keyboardKeys(keys[:])
}

func (p *x11Plugin) sendButton(button uint, press bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.display == nil {
		return errors.New("X server connection closed")
	}
	if button == 0 || button > 9 {
		return errors.New("unsupported pointer button")
	}
	var pressC C.int = C.False
	if press {
		pressC = C.True
	}
	C.XTestFakeButtonEvent(p.display, C.uint(button), pressC, 0)
	C.XFlush(p.display)
	return nil
}

func (p *x11Plugin) PointerButton(button PointerButton, press bool) error {
	if button == PointerButtonLeft {
		return p.sendButton(1, press)
	}
	if button == PointerButtonRight {
		return p.sendButton(3, press)
	}
	if button == PointerButtonMiddle {
		return p.sendButton(2, press)
	}
	return errors.New("unsupported pointer button")
}

func (p *x11Plugin) PointerMove(deltaX, deltaY int) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.display == nil {
		return errors.New("X server connection closed")
	}
	C.XTestFakeRelativeMotionEvent(p.display, C.int(deltaX), C.int(deltaY), 0)
	C.XFlush(p.display)
	return nil
}

func (p *x11Plugin) PointerScroll(deltaHorizontal, deltaVertical int) error {
	p.lock.Lock()
	stepsHorizontal := (p.scrollHorizontal + deltaHorizontal) / scrollDiv
	stepsVertical := (p.scrollVertical + deltaVertical) / scrollDiv
	p.scrollHorizontal = (p.scrollHorizontal + deltaHorizontal) % scrollDiv
	p.scrollVertical = (p.scrollVertical + deltaVertical) % scrollDiv
	p.lock.Unlock()
	var buttonHorizontal uint = 7
	if stepsHorizontal < 0 {
		buttonHorizontal = 6
		stepsHorizontal = -stepsHorizontal
	}
	for i := 0; i < stepsHorizontal; i++ {
		if err := p.sendButton(buttonHorizontal, true); err != nil {
			return err
		}
		if err := p.sendButton(buttonHorizontal, false); err != nil {
			return err
		}
	}
	var buttonVertical uint = 5
	if stepsVertical < 0 {
		buttonVertical = 4
		stepsVertical = -stepsVertical
	}
	for i := 0; i < stepsVertical; i++ {
		if err := p.sendButton(buttonVertical, true); err != nil {
			return err
		}
		if err := p.sendButton(buttonVertical, false); err != nil {
			return err
		}
	}
	return nil
}

func (p *x11Plugin) PointerScrollFinish() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.scrollHorizontal = 0
	p.scrollVertical = 0
	return nil
}
