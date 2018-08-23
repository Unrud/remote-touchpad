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

//go:generate go run keysyms.generator.go

import "errors"

func runeToKeysym(runeValue rune) (int32, error) {
	if runeValue == '\n' {
		runeValue = '\r'
	}
	keysym, found := keysymsMap[runeValue]
	if !found {
		if runeValue < 0x100 || runeValue > 0x10ffff {
			return 0, errors.New("rune not mappend to keysym and " +
				"out of range for direct unicode mapping")
		}
		keysym = 0x01000000 + runeValue
	}
	return keysym, nil
}
