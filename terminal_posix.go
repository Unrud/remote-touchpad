//go:build !windows

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
 *    You should have received a copy of the GNU General Public License
 *    along with Remote-Touchpad.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

// #include <unistd.h>
import "C"
import "os"

func TerminalSupportsColor(fd uintptr) bool {
	return C.isatty(C.int(fd)) != 0
}

func TerminalSetTitle(title string) bool {
	if C.isatty(C.int(os.Stdout.Fd())) != 0 {
		os.Stdout.Write([]byte("\x1b]2;" + title + "\x07"))
		os.Stdout.Sync()
		return true
	}
	return false
}
