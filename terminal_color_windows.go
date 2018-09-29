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

import "syscall"

const (
	enableProcessedOuput            uint32 = 0x1
	enableWrapAtEolOutput           uint32 = 0x2
	enableVirtualTerminalProcessing uint32 = 0x4
)

var (
	kernel32DLL        = syscall.NewLazyDLL("kernel32.dll")
	setConsoleModeProc = kernel32DLL.NewProc("SetConsoleMode")
)

func TerminalSupportsColor(fd uintptr) bool {
	r, _, _ := setConsoleModeProc.Call(fd, uintptr(enableProcessedOuput|
		enableWrapAtEolOutput|enableVirtualTerminalProcessing))
	return r != 0
}
