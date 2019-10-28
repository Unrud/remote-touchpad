/*
 *    Copyright (c) 2018-2019 Unrud <unrud@outlook.com>
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
 *   You should have received a copy of the GNU General Public License
 *   along with Remote-Touchpad.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"syscall"
	"unsafe"
)

const (
	enableProcessedOuput            uint32 = 0x1
	enableWrapAtEolOutput           uint32 = 0x2
	enableVirtualTerminalProcessing uint32 = 0x4
)

var (
	kernel32DLL         = syscall.NewLazyDLL("kernel32.dll")
	setConsoleModeProc  = kernel32DLL.NewProc("SetConsoleMode")
	setConsoleTitleProc = kernel32DLL.NewProc("SetConsoleTitleW")
)

func TerminalSupportsColor(fd uintptr) bool {
	r, _, _ := setConsoleModeProc.Call(fd, uintptr(enableProcessedOuput|
		enableWrapAtEolOutput|enableVirtualTerminalProcessing))
	return r != 0
}

func TerminalSetTitle(title string) bool {
	r, _, _ := setConsoleTitleProc.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))))
	return r != 0
}
