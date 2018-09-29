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

type Keysym int32

const (
	xf86AudioLowerVolume Keysym = 0x1008ff11
	xf86AudioMute        Keysym = 0x1008ff12
	xf86AudioRaiseVolume Keysym = 0x1008ff13
	xf86AudioPlay        Keysym = 0x1008ff14
	xf86AudioPrev        Keysym = 0x1008ff16
	xf86AudioNext        Keysym = 0x1008ff17
)

func RuneToKeysym(runeValue rune) (Keysym, error) {
	if runeValue == '\n' {
		runeValue = '\r'
	}
	keysym, found := keysymsMap[runeValue]
	if !found {
		if runeValue < 0x100 || runeValue > 0x10ffff {
			return 0, errors.New("rune not mappend to keysym and " +
				"out of range for direct unicode mapping")
		}
		keysym = Keysym(0x01000000 + runeValue)
	}
	return keysym, nil
}

func KeyToKeysym(key Key) (Keysym, error) {
	if key == KeyVolumeMute {
		return xf86AudioMute, nil
	}
	if key == KeyVolumeDown {
		return xf86AudioLowerVolume, nil
	}
	if key == KeyVolumeUp {
		return xf86AudioRaiseVolume, nil
	}
	if key == KeyMediaPlayPause {
		return xf86AudioPlay, nil
	}
	if key == KeyMediaPrevTrack {
		return xf86AudioPrev, nil
	}
	if key == KeyMediaNextTrack {
		return xf86AudioNext, nil
	}
	return 0, errors.New("key not mapped to keysym")
}
