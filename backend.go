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

type PointerButton int
type Key int

const (
	PointerButtonLeft PointerButton = iota
	PointerButtonRight
	PointerButtonMiddle
	PointerButtonLimit
)

const (
	KeyVolumeMute Key = iota
	KeyVolumeDown
	KeyVolumeUp
	KeyMediaPlayPause
	KeyMediaPrevTrack
	KeyMediaNextTrack
	KeyBrowserBack
	KeyBrowserForward
	KeySuper
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyBackSpace
	KeyDelete
	KeyReturn
	KeyLimit
)

type BackendInfo struct {
	Name string
	Init func() (Backend, error)
}

var Backends []BackendInfo = []BackendInfo{
	{"X11", InitX11Backend},
	{"RemoteDesktop portal", InitPortalBackend},
	{"Windows", InitWindowsBackend},
}

type UnsupportedPlatformError struct {
	err error
}

func (e UnsupportedPlatformError) Error() string {
	return e.err.Error()
}

type Backend interface {
	Close() error
	KeyboardText(text string) error
	KeyboardKey(key Key) error
	PointerButton(button PointerButton, press bool) error
	PointerMove(deltaX, deltaY int) error
	PointerScroll(deltaHorizontal, deltaVertical int) error
	PointerScrollFinish() error
}
