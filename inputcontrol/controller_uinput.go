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
	"log"
	"os"
	"time"
)

const shiftKeysDelay time.Duration = 50 * time.Millisecond

type uinputController struct {
	keymap   *Keymap
	keyboard uinput.Keyboard
	mouse    uinput.Mouse
}

func init() {
	RegisterController("uinput", InitUinputController, 10)
}

func InitUinputController() (Controller, error) {
	keymapName, keymapSet := os.LookupEnv("REMOTE_TOUCHPAD_UINPUT_KEYMAP")
	if !keymapSet {
		keymapName = "defkeymap"
	}
	keymap, err := LoadKeymap(keymapName)
	if err != nil {
		return nil, err
	}
	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("remote-touchpad-keyboard"))
	if err != nil {
		return nil, err
	}
	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("remote-touchpad-mouse"))
	if err != nil {
		keyboard.Close()
		return nil, err
	}
	if !keymapSet {
		log.Print("Hint: Set the keyboard mapping with the REMOTE_TOUCHPAD_UINPUT_KEYMAP environment variable")
	}
	return &uinputController{keymap, keyboard, mouse}, nil
}

func (p *uinputController) Close() error {
	if err := p.keyboard.Close(); err != nil {
		p.mouse.Close()
		return err
	}
	return p.mouse.Close()
}

func (p *uinputController) KeyboardText(text string) error {
	var activeShiftKeys KeyCombo
	updateShiftKeys := func(keyCombo KeyCombo) error {
		if activeShiftKeys.ShiftKeys == keyCombo.ShiftKeys {
			activeShiftKeys.Key = keyCombo.Key
			return nil
		}
		if activeShiftKeys.Key != 0 {
			time.Sleep(shiftKeysDelay)
			activeShiftKeys.Key = 0
		}
		for i := range keyCombo.ShiftKeys {
			if activeShiftKeys.ShiftKeys[i] != keyCombo.ShiftKeys[i] {
				if activeShiftKeys.ShiftKeys[i] != 0 {
					if err := p.keyboard.KeyUp(activeShiftKeys.ShiftKeys[i]); err != nil {
						return err
					}
					activeShiftKeys.ShiftKeys[i] = 0
				}
				if keyCombo.ShiftKeys[i] != 0 {
					if err := p.keyboard.KeyDown(keyCombo.ShiftKeys[i]); err != nil {
						return err
					}
					activeShiftKeys.ShiftKeys[i] = keyCombo.ShiftKeys[i]
				}
			}
		}
		if keyCombo.Key != 0 {
			activeShiftKeys.Key = keyCombo.Key
			time.Sleep(shiftKeysDelay)
		}
		return nil
	}
	defer updateShiftKeys(KeyCombo{})
	for _, runeValue := range text {
		keyCombo, found := p.keymap.Get(runeValue)
		if !found {
			return fmt.Errorf("unsupported rune: %q", runeValue)
		}
		if err := updateShiftKeys(keyCombo); err != nil {
			return err
		}
		if err := p.keyboard.KeyPress(keyCombo.Key); err != nil {
			return err
		}
	}
	return updateShiftKeys(KeyCombo{Key: 1})
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
