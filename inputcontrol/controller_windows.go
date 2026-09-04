//go:build windows

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

/*
#include <windows.h>
MOUSEINPUT* GetMouseInput(INPUT* input) {
  return input ? &input->mi : NULL;
}
KEYBDINPUT* GetKeyboardInput(INPUT* input) {
  return input ? &input->ki : NULL;
}
HARDWAREINPUT* GetHardwareInput(INPUT* input) {
  return input ? &input->hi : NULL;
}
*/
import "C"

import (
	"fmt"
	"syscall"
	"unsafe"
)

const scrollMult int = 6

type windowsController struct{}

func init() {
	RegisterController("Windows", InitWindowsController, 0)
}

func InitWindowsController() (Controller, error) {
	return &windowsController{}, nil
}

func (p *windowsController) Close() error {
	return nil
}

func (p *windowsController) sendInput(inputs []C.INPUT) error {
	if len(inputs) == 0 {
		return nil
	}
	if sent := C.SendInput(C.UINT(len(inputs)), &inputs[0], C.int(unsafe.Sizeof(inputs[0]))); int(sent) != len(inputs) {
		return fmt.Errorf("SendInput: sent %d of %d inputs: %w", int(sent), len(inputs), syscall.Errno(C.GetLastError()))
	}
	return nil
}

func (p *windowsController) KeyboardText(text string) error {
	inputs := make([]C.INPUT, 0, len(text)*2)
	for _, runeValue := range text {
		input := C.INPUT{_type: C.INPUT_KEYBOARD}
		*C.GetKeyboardInput(&input) = C.KEYBDINPUT{wScan: C.WORD(runeValue), dwFlags: C.KEYEVENTF_UNICODE}
		inputs = append(inputs, input)
		C.GetKeyboardInput(&input).dwFlags |= C.KEYEVENTF_KEYUP
		inputs = append(inputs, input)
	}
	return p.sendInput(inputs)
}

func (p *windowsController) KeyboardKey(key Key) error {
	input := C.INPUT{_type: C.INPUT_KEYBOARD}
	switch key {
	case KeyBackSpace:
		C.GetKeyboardInput(&input).wVk = C.VK_BACK
	case KeyReturn:
		C.GetKeyboardInput(&input).wVk = C.VK_RETURN
	case KeyEnd:
		C.GetKeyboardInput(&input).wVk = C.VK_END
	case KeyHome:
		C.GetKeyboardInput(&input).wVk = C.VK_HOME
	case KeyLeft:
		C.GetKeyboardInput(&input).wVk = C.VK_LEFT
	case KeyUp:
		C.GetKeyboardInput(&input).wVk = C.VK_UP
	case KeyRight:
		C.GetKeyboardInput(&input).wVk = C.VK_RIGHT
	case KeyDown:
		C.GetKeyboardInput(&input).wVk = C.VK_DOWN
	case KeyDelete:
		C.GetKeyboardInput(&input).wVk = C.VK_DELETE
	case KeySuper:
		C.GetKeyboardInput(&input).wVk = C.VK_LWIN
	case KeyBrowserBack:
		C.GetKeyboardInput(&input).wVk = C.VK_BROWSER_BACK
	case KeyBrowserForward:
		C.GetKeyboardInput(&input).wVk = C.VK_BROWSER_FORWARD
	case KeyVolumeMute:
		C.GetKeyboardInput(&input).wVk = C.VK_VOLUME_MUTE
	case KeyVolumeDown:
		C.GetKeyboardInput(&input).wVk = C.VK_VOLUME_DOWN
	case KeyVolumeUp:
		C.GetKeyboardInput(&input).wVk = C.VK_VOLUME_UP
	case KeyMediaNextTrack:
		C.GetKeyboardInput(&input).wVk = C.VK_MEDIA_NEXT_TRACK
	case KeyMediaPrevTrack:
		C.GetKeyboardInput(&input).wVk = C.VK_MEDIA_PREV_TRACK
	case KeyMediaPlayPause:
		C.GetKeyboardInput(&input).wVk = C.VK_MEDIA_PLAY_PAUSE
	default:
		return fmt.Errorf("key not mapped to virtual-key code: %#v", key)
	}
	inputs := []C.INPUT{input, input}
	C.GetKeyboardInput(&inputs[1]).dwFlags |= C.KEYEVENTF_KEYUP
	return p.sendInput(inputs)
}

func (p *windowsController) PointerButton(button PointerButton, press bool) error {
	input := C.INPUT{_type: C.INPUT_MOUSE}
	if button == PointerButtonLeft && press {
		C.GetMouseInput(&input).dwFlags = C.MOUSEEVENTF_LEFTDOWN
	} else if button == PointerButtonLeft {
		C.GetMouseInput(&input).dwFlags = C.MOUSEEVENTF_LEFTUP
	} else if button == PointerButtonMiddle && press {
		C.GetMouseInput(&input).dwFlags = C.MOUSEEVENTF_MIDDLEDOWN
	} else if button == PointerButtonMiddle {
		C.GetMouseInput(&input).dwFlags = C.MOUSEEVENTF_MIDDLEUP
	} else if button == PointerButtonRight && press {
		C.GetMouseInput(&input).dwFlags = C.MOUSEEVENTF_RIGHTDOWN
	} else if button == PointerButtonRight {
		C.GetMouseInput(&input).dwFlags = C.MOUSEEVENTF_RIGHTUP
	} else {
		return fmt.Errorf("unsupported pointer button: %#v", button)
	}
	return p.sendInput([]C.INPUT{input})
}

func (p *windowsController) PointerMove(deltaX, deltaY int) error {
	input := C.INPUT{_type: C.INPUT_MOUSE}
	*C.GetMouseInput(&input) = C.MOUSEINPUT{
		dx:      C.LONG(deltaX),
		dy:      C.LONG(deltaY),
		dwFlags: C.MOUSEEVENTF_MOVE,
	}
	return p.sendInput([]C.INPUT{input})
}

func (p *windowsController) PointerScroll(deltaHorizontal, deltaVertical int, finish bool) error {
	inputs := make([]C.INPUT, 0, 2)
	if deltaHorizontal != 0 {
		input := C.INPUT{_type: C.INPUT_MOUSE}
		*C.GetMouseInput(&input) = C.MOUSEINPUT{
			dwFlags:   C.MOUSEEVENTF_HWHEEL,
			mouseData: C.DWORD(deltaHorizontal * scrollMult),
		}
		inputs = append(inputs, input)
	}
	if deltaVertical != 0 {
		input := C.INPUT{_type: C.INPUT_MOUSE}
		*C.GetMouseInput(&input) = C.MOUSEINPUT{
			dwFlags:   C.MOUSEEVENTF_WHEEL,
			mouseData: C.DWORD(-deltaVertical * scrollMult),
		}
		inputs = append(inputs, input)
	}
	return p.sendInput(inputs)
}
