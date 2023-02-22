//go:build uinput

/*
 *    Copyright (c) 2023 De_Coder github.com/ps100000
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

import "github.com/bendahl/uinput"
import "errors"

var ukeysMap = map[Keysym]int {
	0x0061: uinput.KeyA,
	0x0062: uinput.KeyB,
	0x0063: uinput.KeyC,
	0x0064: uinput.KeyD,
	0x0065: uinput.KeyE,
	0x0066: uinput.KeyF,
	0x0067: uinput.KeyG,
	0x0068: uinput.KeyH,
	0x0069: uinput.KeyI,
	0x006a: uinput.KeyJ,
	0x006b: uinput.KeyK,
	0x006c: uinput.KeyL,
	0x006d: uinput.KeyM,
	0x006e: uinput.KeyN,
	0x006f: uinput.KeyO,
	0x0070: uinput.KeyP,
	0x0071: uinput.KeyQ,
	0x0072: uinput.KeyR,
	0x0073: uinput.KeyS,
	0x0074: uinput.KeyT,
	0x0075: uinput.KeyU,
	0x0076: uinput.KeyV,
	0x0077: uinput.KeyW,
	0x0078: uinput.KeyX,
	0x0079: uinput.KeyY,
	0x007a: uinput.KeyZ,

	0x0030: uinput.Key0,
	0x0031: uinput.Key1,
	0x0032: uinput.Key2,
	0x0033: uinput.Key3,
	0x0034: uinput.Key4,
	0x0035: uinput.Key5,
	0x0036: uinput.Key6,
	0x0037: uinput.Key7,
	0x0038: uinput.Key8,
	0x0039: uinput.Key9,

	0xff1b: uinput.KeyEsc,
	0x002d: uinput.KeyMinus,
	0x002b: uinput.KeyKpplus,
	0x002a: uinput.KeyKpasterisk,
	0x003d: uinput.KeyEqual,
	0xff08: uinput.KeyBackspace,
	0xff09: uinput.KeyTab,
	0x0028: uinput.KeyLeftbrace,
	0x0029: uinput.KeyRightbrace,
	0xff0d: uinput.KeyEnter,
	0x003b: uinput.KeySemicolon,
	0x0027: uinput.KeyApostrophe,
	0x0060: uinput.KeyGrave,
	0x005c: uinput.KeyBackslash,
	0x002c: uinput.KeyComma,
	0x0abd: uinput.KeyDot,
	0x002f: uinput.KeySlash,
	0x0020: uinput.KeySpace,
	0xff50: uinput.KeyHome,
	0xff52: uinput.KeyUp,
	0xff51: uinput.KeyLeft,
	0xff53: uinput.KeyRight,
	0xff57: uinput.KeyEnd,
	0xff54: uinput.KeyDown,
	0xffff: uinput.KeyDelete,
	0x1008FF12: uinput.KeyMute,
	0x1008FF11: uinput.KeyVolumedown,
	0x1008FF13: uinput.KeyVolumeup,
	0xffeb: uinput.KeyLeftmeta,
	0x1008ff26: uinput.KeyBack,
	0x1008ff27: uinput.KeyForward,
	0x1008ff17: uinput.KeyNextsong,
	0x1008ff14: uinput.KeyPlaypause,
	0x1008ff16: uinput.KeyPrevioussong,
}

type uinputController struct {
	keyboard uinput.Keyboard
	mouse uinput.Mouse
}

func init() {
	RegisterController("Uinput", InitUinputController, 0)
}

func InitUinputController() (Controller, error) {
	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("remote-tp-keyboard"))
	if err != nil {
		return nil, err
	}
	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("remote-tp-mouse"))
	if err != nil {
		keyboard.Close()
		return nil, err
	}
	p := &uinputController{keyboard, mouse}
	return p, nil
}

func (p *uinputController) Close() error {
	p.keyboard.Close()
	p.mouse.Close()
	return nil
}

func (p *uinputController) sendKeysym(sym Keysym) error {
	if sym >= 0x0041 && sym <= 0x005a { // capital letters
		p.keyboard.KeyDown(uinput.KeyLeftshift)
		p.keyboard.KeyPress(ukeysMap[sym + 0x20])
		p.keyboard.KeyUp(uinput.KeyLeftshift)
	} else if key, ok := ukeysMap[sym]; ok {
		p.keyboard.KeyPress(key)
	}
	return nil
}

func (p *uinputController) KeyboardText(text string) error {
	for _, runeValue := range text {
		keysym, err := RuneToKeysym(runeValue)
		if err != nil {
			return err
		}
		err = p.sendKeysym(keysym);

	}
	return nil
}

func (p *uinputController) KeyboardKey(key Key) error {
	keysym, err := KeyToKeysym(key)
	if err != nil {
		return err
	}
	return p.sendKeysym(keysym)
}

func (p *uinputController) PointerButton(button PointerButton, press bool) error {
	switch button {
		case PointerButtonLeft:
		if press {
			p.mouse.LeftPress()
		} else {
			p.mouse.LeftRelease()
		}
		case PointerButtonRight:
		if press {
			p.mouse.RightPress()
		} else {
			p.mouse.RightRelease()
		}
		case PointerButtonMiddle:
		if press {
			p.mouse.MiddlePress()
		} else {
			p.mouse.MiddleRelease()
		}
		default:
		return errors.New("unsupported pointer button")
	}
	return nil
}

func (p *uinputController) PointerMove(deltaX, deltaY int) error {
	p.mouse.Move(int32(deltaX), int32(deltaY))
	return nil
}

func (p *uinputController) PointerScroll(deltaHorizontal, deltaVertical int, finish bool) error {
	p.mouse.Wheel(false, int32(deltaVertical))
	p.mouse.Wheel(true, int32(deltaHorizontal))
	return nil
}
