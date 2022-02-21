// +build windows

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

package main

import (
	"errors"
	"syscall"
	"unsafe"
)

const (
	inputMouse    uintptr = 0x0
	inputKeyboard uintptr = 0x1

	keyeventfKeyup   uint32 = 0x2
	keyeventfUnicode uint32 = 0x4

	vkBack           uint16 = 0x8
	vkReturn         uint16 = 0xD
	vkEnd            uint16 = 0x23
	vkHome           uint16 = 0x24
	vkLeft           uint16 = 0x25
	vkUp             uint16 = 0x26
	vkRight          uint16 = 0x27
	vkDown           uint16 = 0x28
	vkDelete         uint16 = 0x2E
	vkLwin           uint16 = 0x5B
	vkBrowserBack    uint16 = 0xA6
	vkBrowserForward uint16 = 0xA7
	vkVolumeMute     uint16 = 0xAD
	vkVolumeDown     uint16 = 0xAE
	vkVolumeUp       uint16 = 0xAF
	vkMediaNextTrack uint16 = 0xB0
	vkMediaPrevTrack uint16 = 0xB1
	vkMediaPlayPause uint16 = 0xB3

	mouseeventfMove       uint32 = 0x1
	mouseeventfLeftdown   uint32 = 0x2
	mouseeventfLeftup     uint32 = 0x4
	mouseeventfRightdown  uint32 = 0x8
	mouseeventfRightup    uint32 = 0x10
	mouseeventfMiddledown uint32 = 0x20
	mouseeventfMiddleup   uint32 = 0x40
	mouseeventfWheel      uint32 = 0x800
	mouseeventfHwheel     uint32 = 0x1000

	scrollMult int = 6
)

var (
	user32DLL     = syscall.NewLazyDLL("user32.dll")
	sendInputProc = user32DLL.NewProc("SendInput")
)

type mouseInput struct {
	typ uintptr // HACK: padded uint32

	dx, dy                   int32
	mouseData, dwFlags, time uint32
	dwExtraInfo              uintptr
}

type keybdInput struct {
	typ uintptr // HACK: padded uint32

	wVk, wScan    uint16
	dwFlags, time uint32
	dwExtraInfo   uintptr

	padding [8]byte
}

type windowsBackend struct{}

func init() {
	RegisterBackend("Windows", InitWindowsBackend, 0)
}

func InitWindowsBackend() (Backend, error) {
	p := &windowsBackend{}
	if err := sendInputProc.Find(); err != nil {
		return nil, UnsupportedPlatformError{err}
	}
	return p, nil
}

func (p *windowsBackend) Close() error {
	return nil
}

func (p *windowsBackend) sendInput(inputs []keybdInput) error {
	if len(inputs) == 0 {
		return nil
	}
	if r, _, err := sendInputProc.Call(uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0])); int(r) != len(inputs) {
		return err
	}
	return nil
}

func (p *windowsBackend) KeyboardText(text string) error {
	inputs := make([]keybdInput, 0, len(text)*2)
	for _, runeValue := range text {
		in := keybdInput{typ: inputKeyboard, wScan: uint16(runeValue), dwFlags: keyeventfUnicode}
		inputs = append(inputs, in)
		in.dwFlags |= keyeventfKeyup
		inputs = append(inputs, in)
	}
	if len(inputs) == 0 {
		return nil
	}
	return p.sendInput(inputs)
}

func (p *windowsBackend) KeyboardKey(key Key) error {
	input := keybdInput{typ: inputKeyboard}
	if key == KeyBackSpace {
		input.wVk = vkBack
	} else if key == KeyReturn {
		input.wVk = vkReturn
	} else if key == KeyEnd {
		input.wVk = vkEnd
	} else if key == KeyHome {
		input.wVk = vkHome
	} else if key == KeyLeft {
		input.wVk = vkLeft
	} else if key == KeyUp {
		input.wVk = vkUp
	} else if key == KeyRight {
		input.wVk = vkRight
	} else if key == KeyDown {
		input.wVk = vkDown
	} else if key == KeyDelete {
		input.wVk = vkDelete
	} else if key == KeySuper {
		input.wVk = vkLwin
	} else if key == KeyBrowserBack {
		input.wVk = vkBrowserBack
	} else if key == KeyBrowserForward {
		input.wVk = vkBrowserForward
	} else if key == KeyVolumeMute {
		input.wVk = vkVolumeMute
	} else if key == KeyVolumeDown {
		input.wVk = vkVolumeDown
	} else if key == KeyVolumeUp {
		input.wVk = vkVolumeUp
	} else if key == KeyMediaNextTrack {
		input.wVk = vkMediaNextTrack
	} else if key == KeyMediaPrevTrack {
		input.wVk = vkMediaPrevTrack
	} else if key == KeyMediaPlayPause {
		input.wVk = vkMediaPlayPause
	} else {
		return errors.New("key not mapped to virtual-key code")
	}
	inputs := [...]keybdInput{input, input}
	inputs[1].dwFlags |= keyeventfKeyup
	return p.sendInput(inputs[:])
}

func (p *windowsBackend) PointerButton(button PointerButton, press bool) error {
	input := mouseInput{typ: inputMouse}
	if button == PointerButtonLeft && press {
		input.dwFlags = mouseeventfLeftdown
	} else if button == PointerButtonLeft {
		input.dwFlags = mouseeventfLeftup
	} else if button == PointerButtonMiddle && press {
		input.dwFlags = mouseeventfMiddledown
	} else if button == PointerButtonMiddle {
		input.dwFlags = mouseeventfMiddleup
	} else if button == PointerButtonRight && press {
		input.dwFlags = mouseeventfRightdown
	} else if button == PointerButtonRight {
		input.dwFlags = mouseeventfRightup
	} else {
		return errors.New("unsupported pointer button")
	}
	if r, _, err := sendInputProc.Call(1, uintptr(unsafe.Pointer(&input)),
		unsafe.Sizeof(input)); int(r) != 1 {
		return err
	}
	return nil
}

func (p *windowsBackend) PointerMove(deltaX, deltaY int) error {
	input := mouseInput{
		typ:     inputMouse,
		dx:      int32(deltaX),
		dy:      int32(deltaY),
		dwFlags: mouseeventfMove,
	}
	if r, _, err := sendInputProc.Call(1, uintptr(unsafe.Pointer(&input)),
		unsafe.Sizeof(input)); int(r) != 1 {
		return err
	}
	return nil
}

func (p *windowsBackend) PointerScroll(deltaHorizontal, deltaVertical int, finish bool) error {
	inputs := make([]mouseInput, 0, 2)
	if deltaHorizontal != 0 {
		inputs = append(inputs, mouseInput{
			typ:       inputMouse,
			dwFlags:   mouseeventfHwheel,
			mouseData: uint32(deltaHorizontal * scrollMult),
		})
	}
	if deltaVertical != 0 {
		inputs = append(inputs, mouseInput{
			typ:       inputMouse,
			dwFlags:   mouseeventfWheel,
			mouseData: uint32(deltaVertical * scrollMult),
		})
	}
	if len(inputs) == 0 {
		return nil
	}
	if r, _, err := sendInputProc.Call(uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0])); int(r) != len(inputs) {
		return err
	}
	return nil
}
