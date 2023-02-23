//go:build portal || x11

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

//go:generate go run keysyms.generator.go

import "fmt"

type Keysym int32

const (
	// X11/keysymdef.h
	xkBackSpace Keysym = 0xff08
	xkReturn    Keysym = 0xff0D
	xkDelete    Keysym = 0xffff
	xkHome      Keysym = 0xff50
	xkLeft      Keysym = 0xff51
	xkUp        Keysym = 0xff52
	xkRight     Keysym = 0xff53
	xkDown      Keysym = 0xff54
	xkEnd       Keysym = 0xff57
	xkSuperL    Keysym = 0xffeb
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
			return 0, fmt.Errorf("rune not mappend to keysym and "+
				"out of range for direct unicode mapping: %q", runeValue)
		}
		keysym = Keysym(0x01000000 + runeValue)
	}
	return keysym, nil
}

func KeyToKeysym(key Key) (Keysym, error) {
	switch key {
	case KeyBackSpace:
		return xkBackSpace, nil
	case KeyReturn:
		return xkReturn, nil
	case KeyDelete:
		return xkDelete, nil
	case KeyHome:
		return xkHome, nil
	case KeyLeft:
		return xkLeft, nil
	case KeyUp:
		return xkUp, nil
	case KeyRight:
		return xkRight, nil
	case KeyDown:
		return xkDown, nil
	case KeyEnd:
		return xkEnd, nil
	case KeySuper:
		return xkSuperL, nil
	case KeyVolumeMute:
		return xf86xkAudioMute, nil
	case KeyVolumeDown:
		return xf86xkAudioLowerVolume, nil
	case KeyVolumeUp:
		return xf86xkAudioRaiseVolume, nil
	case KeyMediaPlayPause:
		return xf86xkAudioPlay, nil
	case KeyMediaPrevTrack:
		return xf86xkAudioPrev, nil
	case KeyMediaNextTrack:
		return xf86xkAudioNext, nil
	case KeyBrowserBack:
		return xf86xkBack, nil
	case KeyBrowserForward:
		return xf86xkForward, nil
	default:
		return 0, fmt.Errorf("key not mapped to keysym: %#v", key)
	}
}
