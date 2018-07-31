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
import "C"
import (
	"errors"
	"sync"
	"time"
	"unsafe"
)

const typingDelay time.Duration = 100 * time.Millisecond

type x11Plugin struct {
	display *C.Display
	lock    sync.Mutex
}

func InitX11Plugin() (Plugin, error) {
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

func (p *x11Plugin) KeyboardText(text string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.display == nil {
		return errors.New("X server connection closed")
	}
	if text == "" {
		return nil
	}
	var minKeycodes, maxKeycodes C.int
	C.XDisplayKeycodes(p.display, &minKeycodes, &maxKeycodes)
	var keysymsPerKeycode C.int
	keysyms := C.XGetKeyboardMapping(p.display, C.KeyCode(minKeycodes),
		maxKeycodes-minKeycodes+1, &keysymsPerKeycode)
	if keysyms == nil {
		return errors.New("failed to get keyboard mapping")
	}
	defer C.XFree(unsafe.Pointer(keysyms))
	emptyKeycode := C.KeyCode(0)
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
		emptyKeycode = keycode
		break
	}
	if emptyKeycode == 0 {
		return errors.New("no empty keycode found")
	}
	keycodeMapping := make([]C.KeySym, keysymsPerKeycode)
	defer func() {
		for i := range keycodeMapping {
			keycodeMapping[i] = 0
		}
		C.XChangeKeyboardMapping(p.display, C.int(emptyKeycode), keysymsPerKeycode,
			(*C.KeySym)(unsafe.Pointer(&keycodeMapping[0])), 1)
		C.XFlush(p.display)
	}()
	for _, runeValue := range text {
		keysym := C.KeySym(0x01000000 + runeValue)
		for i := range keycodeMapping {
			keycodeMapping[i] = keysym
		}
		C.XChangeKeyboardMapping(p.display, C.int(emptyKeycode), keysymsPerKeycode,
			(*C.KeySym)(unsafe.Pointer(&keycodeMapping[0])), 1)
		C.XTestFakeKeyEvent(p.display, C.uint(emptyKeycode), C.True, 0)
		C.XTestFakeKeyEvent(p.display, C.uint(emptyKeycode), C.False, 0)
		// race condition!
		C.XFlush(p.display)
		time.Sleep(typingDelay)
	}
	return nil
}

func (p *x11Plugin) PointerButton(button uint, press bool) error {
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

func (p *x11Plugin) PointerScroll(stepsHorizontal, stepsVertical int) error {
	var buttonHorizontal uint = 7
	if stepsHorizontal < 0 {
		buttonHorizontal = 6
		stepsHorizontal = -stepsHorizontal
	}
	for i := 0; i < stepsHorizontal; i++ {
		if err := p.PointerButton(buttonHorizontal, true); err != nil {
			return err
		}
		if err := p.PointerButton(buttonHorizontal, false); err != nil {
			return err
		}
	}
	var buttonVertical uint = 5
	if stepsVertical < 0 {
		buttonVertical = 4
		stepsVertical = -stepsVertical
	}
	for i := 0; i < stepsVertical; i++ {
		if err := p.PointerButton(buttonVertical, true); err != nil {
			return err
		}
		if err := p.PointerButton(buttonVertical, false); err != nil {
			return err
		}
	}
	return nil
}
