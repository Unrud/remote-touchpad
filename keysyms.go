// +build portal x11

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

//go:generate go run keysyms.generator.go

import "errors"

type Keysym int32

const (
	// X11/keysymdef.h
	xkSuperL Keysym = 0xffeb
	// X11/XF86keysym.h
	xf86xkAudioLowerVolume Keysym = 0x1008ff11
	xf86xkAudioMute        Keysym = 0x1008ff12
	xf86xkAudioRaiseVolume Keysym = 0x1008ff13
	xf86xkAudioPlay        Keysym = 0x1008ff14
	xf86xkAudioPrev        Keysym = 0x1008ff16
	xf86xkAudioNext        Keysym = 0x1008ff17
	xf86xkBack             Keysym = 0x1008ff26
	xf86xkForward          Keysym = 0x1008ff27
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
	if key == KeySuper {
		return xkSuperL, nil
	}
	if key == KeyVolumeMute {
		return xf86xkAudioMute, nil
	}
	if key == KeyVolumeDown {
		return xf86xkAudioLowerVolume, nil
	}
	if key == KeyVolumeUp {
		return xf86xkAudioRaiseVolume, nil
	}
	if key == KeyMediaPlayPause {
		return xf86xkAudioPlay, nil
	}
	if key == KeyMediaPrevTrack {
		return xf86xkAudioPrev, nil
	}
	if key == KeyMediaNextTrack {
		return xf86xkAudioNext, nil
	}
	if key == KeyBrowserBack {
		return xf86xkBack, nil
	}
	if key == KeyBrowserForward {
		return xf86xkForward, nil
	}
	return 0, errors.New("key not mapped to keysym")
}
