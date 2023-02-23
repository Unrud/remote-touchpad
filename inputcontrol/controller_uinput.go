//go:build uinput

/*
 *    Copyright (c) 2023 De_Coder github.com/ps100000
 *    Copyright (c) 2023 Unrud <unrud@outlook.com>
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

import (
	"fmt"
	"github.com/bendahl/uinput"
	"unicode"
)

var ukeysMap = map[rune]int{
	'a':  uinput.KeyA,
	'b':  uinput.KeyB,
	'c':  uinput.KeyC,
	'd':  uinput.KeyD,
	'e':  uinput.KeyE,
	'f':  uinput.KeyF,
	'g':  uinput.KeyG,
	'h':  uinput.KeyH,
	'i':  uinput.KeyI,
	'j':  uinput.KeyJ,
	'k':  uinput.KeyK,
	'l':  uinput.KeyL,
	'm':  uinput.KeyM,
	'n':  uinput.KeyN,
	'o':  uinput.KeyO,
	'p':  uinput.KeyP,
	'q':  uinput.KeyQ,
	'r':  uinput.KeyR,
	's':  uinput.KeyS,
	't':  uinput.KeyT,
	'u':  uinput.KeyU,
	'v':  uinput.KeyV,
	'w':  uinput.KeyW,
	'x':  uinput.KeyX,
	'y':  uinput.KeyY,
	'z':  uinput.KeyZ,
	'0':  uinput.Key0,
	'1':  uinput.Key1,
	'2':  uinput.Key2,
	'3':  uinput.Key3,
	'4':  uinput.Key4,
	'5':  uinput.Key5,
	'6':  uinput.Key6,
	'7':  uinput.Key7,
	'8':  uinput.Key8,
	'9':  uinput.Key9,
	0x1b: uinput.KeyEsc,
	'-':  uinput.KeyMinus,
	'+':  uinput.KeyKpplus,
	'*':  uinput.KeyKpasterisk,
	'=':  uinput.KeyEqual,
	0x08: uinput.KeyBackspace,
	'\t': uinput.KeyTab,
	'(':  uinput.KeyLeftbrace,
	')':  uinput.KeyRightbrace,
	'\n': uinput.KeyEnter,
	';':  uinput.KeySemicolon,
	'\'': uinput.KeyApostrophe,
	'`':  uinput.KeyGrave,
	'\\': uinput.KeyBackslash,
	',':  uinput.KeyComma,
	'.':  uinput.KeyDot,
	'/':  uinput.KeySlash,
	' ':  uinput.KeySpace,
}

type uinputController struct {
	keyboard uinput.Keyboard
	mouse    uinput.Mouse
}

func init() {
	RegisterController("uinput", InitUinputController, 0)
}

func InitUinputController() (Controller, error) {
	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("remote-touchpad-keyboard"))
	if err != nil {
		return nil, err
	}
	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("remote-touchpad-mouse"))
	if err != nil {
		keyboard.Close()
		return nil, err
	}
	return &uinputController{keyboard, mouse}, nil
}

func (p *uinputController) Close() error {
	if err := p.keyboard.Close(); err != nil {
		p.mouse.Close()
		return err
	}
	return p.mouse.Close()
}

// No support for keyboard layouts, only works with QWERTY
func (p *uinputController) KeyboardText(text string) error {
	for _, runeValue := range text {
		modifierShift := false
		if unicode.IsUpper(runeValue) {
			modifierShift = true
			runeValue = unicode.ToLower(runeValue)
		}
		uinputKey, found := ukeysMap[runeValue]
		if !found {
			return fmt.Errorf("unsupported rune: %q", runeValue)
		}
		if modifierShift {
			if err := p.keyboard.KeyDown(uinput.KeyLeftshift); err != nil {
				return err
			}
		}
		if err := p.keyboard.KeyPress(uinputKey); err != nil {
			return err
		}
		if modifierShift {
			if err := p.keyboard.KeyUp(uinput.KeyLeftshift); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *uinputController) KeyboardKey(key Key) error {
	var uinputKey int
	switch key {
	case KeyBackSpace:
		uinputKey = uinput.KeyBackspace
	case KeyReturn:
		uinputKey = uinput.KeyEnter
	case KeyDelete:
		uinputKey = uinput.KeyDelete
	case KeyHome:
		uinputKey = uinput.KeyHome
	case KeyLeft:
		uinputKey = uinput.KeyLeft
	case KeyUp:
		uinputKey = uinput.KeyUp
	case KeyRight:
		uinputKey = uinput.KeyRight
	case KeyDown:
		uinputKey = uinput.KeyDown
	case KeyEnd:
		uinputKey = uinput.KeyEnd
	case KeySuper:
		uinputKey = uinput.KeyLeftmeta
	case KeyVolumeMute:
		uinputKey = uinput.KeyMute
	case KeyVolumeDown:
		uinputKey = uinput.KeyVolumedown
	case KeyVolumeUp:
		uinputKey = uinput.KeyVolumeup
	case KeyMediaPlayPause:
		uinputKey = uinput.KeyPlaypause
	case KeyMediaPrevTrack:
		uinputKey = uinput.KeyPrevioussong
	case KeyMediaNextTrack:
		uinputKey = uinput.KeyNextsong
	case KeyBrowserBack:
		uinputKey = uinput.KeyBack
	case KeyBrowserForward:
		uinputKey = uinput.KeyForward
	default:
		return fmt.Errorf("unsupported key: %#v", key)
	}
	return p.keyboard.KeyPress(uinputKey)
}

func (p *uinputController) PointerButton(button PointerButton, press bool) error {
	switch {
	case button == PointerButtonLeft && press:
		return p.mouse.LeftPress()
	case button == PointerButtonLeft && !press:
		return p.mouse.LeftRelease()
	case button == PointerButtonRight && press:
		return p.mouse.RightPress()
	case button == PointerButtonRight && !press:
		return p.mouse.RightRelease()
	case button == PointerButtonMiddle && press:
		return p.mouse.MiddlePress()
	case button == PointerButtonMiddle && !press:
		return p.mouse.MiddleRelease()
	default:
		return fmt.Errorf("unsupported pointer button: %#v", button)
	}
}

func (p *uinputController) PointerMove(deltaX, deltaY int) error {
	return p.mouse.Move(int32(deltaX), int32(deltaY))
}

func (p *uinputController) PointerScroll(deltaHorizontal, deltaVertical int, finish bool) error {
	if err := p.mouse.Wheel(false, int32(deltaVertical)); err != nil {
		return err
	}
	return p.mouse.Wheel(true, int32(deltaHorizontal))
}
