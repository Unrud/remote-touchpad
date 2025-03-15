//go:build x11

/*
 *    Copyright (c) 2018 Unrud <unrud@outlook.com>
 *
 *    This file is part of Remote-Touchpad.
 *
 *    Remote-Touchpad is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU General Public License as published by
 *    the Free Software Foundation, either version 3 of the License, or
 *    (at your option) any later version.
 *
 *    Remote-Touchpad is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU General Public License for more details.
 *
 *    You should have received a copy of the GNU General Public License
 *    along with Remote-Touchpad.  If not, see <http://www.gnu.org/licenses/>.
 */

package inputcontrol

// #cgo LDFLAGS: -lX11 -lXrandr -lXtst
// #include <stdlib.h>
// #include <X11/Xlib.h>
// #include <X11/Intrinsic.h>
// #include <X11/extensions/Xrandr.h>
// #include <X11/extensions/XTest.h>
// #include <X11/XKBlib.h>
// Window MacroDefaultRootWindow(Display *dpy) {
//     return DefaultRootWindow(dpy);
// }
import "C"
import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	keyboardMappingDelay time.Duration = 500 * time.Millisecond
	scrollDiv            int           = 20
)

var modifierIndices [6]uint = [...]uint{C.ShiftMapIndex, C.Mod1MapIndex,
	C.Mod2MapIndex, C.Mod3MapIndex, C.Mod4MapIndex, C.Mod5MapIndex}

type x11Controller struct {
	display                          *C.Display
	lock                             sync.Mutex
	scrollHorizontal, scrollVertical int
}

func init() {
	RegisterController("X11", InitX11Controller, 0)
}

func InitX11Controller(saveRestoreToken bool) (Controller, error) {
	display := C.XOpenDisplay(nil)
	if display == nil {
		return nil, &UnsupportedPlatformError{
			errors.New("failed to connect to X server")}
	}
	p := &x11Controller{display: display}
	if p.xIsXwayland() {
		p.Close()
		return nil, &UnsupportedPlatformError{
			errors.New("X server is Xwayland")}
	}
	return p, nil
}

func (p *x11Controller) xIsXwayland() bool {
	// Detection method from https://gitlab.freedesktop.org/xorg/app/xisxwayland/-/blob/xisxwayland-2/xisxwayland.c
	var opcode, event, error, major, minor C.int
	xwaylandExtensionName := C.CString("XWAYLAND")
	defer C.free(unsafe.Pointer(xwaylandExtensionName))
	if C.XQueryExtension(p.display, xwaylandExtensionName, &opcode, &event, &error) != 0 {
		return true
	}
	if C.XRRQueryExtension(p.display, &event, &error) == 0 ||
		C.XRRQueryVersion(p.display, &major, &minor) == 0 {
		return false
	}
	resources := C.XRRGetScreenResourcesCurrent(p.display, C.MacroDefaultRootWindow(p.display))
	if resources == nil {
		return false
	}
	defer C.XRRFreeScreenResources(resources)
	if resources.noutput < 1 {
		return false
	}
	output := C.XRRGetOutputInfo(p.display, resources, *resources.outputs)
	if output == nil {
		return false
	}
	defer C.XRRFreeOutputInfo(output)
	return strings.HasPrefix(C.GoString(output.name), "XWAYLAND")
}

func (p *x11Controller) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.display == nil {
		return errors.New("X server connection closed")
	}
	C.XCloseDisplay(p.display)
	p.display = nil
	return nil
}

func (p *x11Controller) findEmptyKeycodeLocked() (C.KeyCode, C.int, error) {
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

func (p *x11Controller) changeKeyMappingLocked(keysymsPerKeycode C.int,
	keycode C.KeyCode, keysym Keysym) {
	keycodeMapping := make([]C.KeySym, keysymsPerKeycode)
	for i := range keycodeMapping {
		keycodeMapping[i] = C.KeySym(keysym)
	}
	C.XChangeKeyboardMapping(p.display, C.int(keycode), keysymsPerKeycode,
		(*C.KeySym)(unsafe.Pointer(&keycodeMapping[0])), 1)
	C.XFlush(p.display)
}

func (p *x11Controller) getModKeycodesLocked() map[uint]C.KeyCode {
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

func (p *x11Controller) findKeycodeLocked(keyboard C.XkbDescPtr,
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

func (p *x11Controller) sendModsLocked(modKeycodes map[uint]C.KeyCode, mods C.uint,
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

func (p *x11Controller) keyboardKeys(keys []Keysym) error {
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

func (p *x11Controller) KeyboardText(text string) error {
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

func (p *x11Controller) KeyboardKey(key Key) error {
	keysym, err := KeyToKeysym(key)
	if err != nil {
		return err
	}
	keys := [...]Keysym{keysym}
	return p.keyboardKeys(keys[:])
}

func (p *x11Controller) sendButton(button uint, press bool) error {
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

func (p *x11Controller) PointerButton(button PointerButton, press bool) error {
	switch button {
	case PointerButtonLeft:
		return p.sendButton(1, press)
	case PointerButtonRight:
		return p.sendButton(3, press)
	case PointerButtonMiddle:
		return p.sendButton(2, press)
	default:
		return fmt.Errorf("unsupported pointer button: %#v", button)
	}
}

func (p *x11Controller) PointerMove(deltaX, deltaY int) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.display == nil {
		return errors.New("X server connection closed")
	}
	C.XTestFakeRelativeMotionEvent(p.display, C.int(deltaX), C.int(deltaY), 0)
	C.XFlush(p.display)
	return nil
}

func (p *x11Controller) PointerScroll(deltaHorizontal, deltaVertical int, finish bool) error {
	p.lock.Lock()
	stepsHorizontal := (p.scrollHorizontal + deltaHorizontal) / scrollDiv
	stepsVertical := (p.scrollVertical + deltaVertical) / scrollDiv
	if finish {
		p.scrollHorizontal = 0
		p.scrollVertical = 0
	} else {
		p.scrollHorizontal = (p.scrollHorizontal + deltaHorizontal) % scrollDiv
		p.scrollVertical = (p.scrollVertical + deltaVertical) % scrollDiv
	}
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
