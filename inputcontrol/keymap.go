//go:build uinput

/*
 *    Copyright (c) 2023 Unrud <unrud@outlook.com>
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

import (
	"fmt"
	"os/exec"
	"strings"
	"unsafe"
)

const bkeymapFileSignature = "bkeymap"

// uapi/linux/keyboard.h
const (
	nrKeys       = 128
	maxNrKeymaps = 256
	nrShift      = 9

	ktLatin  = 0
	ktShift  = 7
	ktLetter = 11
)

type KeyCombo struct {
	Key       int
	ShiftKeys [nrShift]int
}

type Keymap struct {
	keyComboMap map[rune]KeyCombo
}

func (keymap Keymap) Get(character rune) (keyCombo KeyCombo, found bool) {
	keyCombo, found = keymap.keyComboMap[character]
	return
}

func LoadKeymap(name string) (*Keymap, error) {
	if len(name) == 0 || strings.HasPrefix(name, "-") {
		return nil, fmt.Errorf("invalid keymap name: %#v", name)
	}
	bkeymap, err := exec.Command("loadkeys", "--bkeymap", name).Output()
	if err != nil {
		return nil, &UnsupportedPlatformError{err}
	}
	if !strings.HasPrefix(string(bkeymap), bkeymapFileSignature) {
		return nil, fmt.Errorf("invalid bkeymap: signature not found")
	}
	bkeymapHeader := bkeymap[len(bkeymapFileSignature):]
	var keymaps [][nrKeys]int
	if len(bkeymapHeader) < maxNrKeymaps {
		return nil, fmt.Errorf("invalid bkeymap: EOF")
	}
	currentBkeymap := bkeymapHeader[maxNrKeymaps:]
	for keymapIndex := 0; keymapIndex < maxNrKeymaps; keymapIndex += 1 {
		if bkeymapHeader[keymapIndex] == 0 {
			continue
		}
		if len(currentBkeymap) < nrKeys*2 {
			return nil, fmt.Errorf("invalid bkeymap: EOF")
		}
		for len(keymaps) <= keymapIndex {
			keymaps = append(keymaps, [nrKeys]int{})
		}
		for key := 0; key < nrKeys; key += 1 {
			var keysym uint16
			copy(unsafe.Slice((*byte)(unsafe.Pointer(&keysym)), 2),
				currentBkeymap[key*2:])
			keymaps[keymapIndex][key] = int(keysym)
		}
		currentBkeymap = currentBkeymap[nrKeys*2:]
	}
	keymap := Keymap{keyComboMap: make(map[rune]KeyCombo)}
	if len(keymaps) == 0 {
		return &keymap, nil
	}
	var shiftKeys [nrShift]int
	for key, keysym := range keymaps[0] {
		keyType := keysym >> 8
		keyValue := keysym & 0xff
		if keyType == ktShift && keyValue < nrShift && shiftKeys[keyValue] == 0 {
			shiftKeys[keyValue] = key
		}
	}
keymapLoop:
	for keymapIndex := range keymaps {
		var activeShiftKeys [nrShift]int
		for i, shiftKey := range shiftKeys {
			if keymapIndex&(1<<i) != 0 {
				if shiftKey == 0 {
					continue keymapLoop
				}
				activeShiftKeys[i] = shiftKey
			}
		}
		for key, keysym := range keymaps[keymapIndex] {
			keyType := keysym >> 8
			keyValue := keysym & 0xff
			if (keyType != ktLatin && keyType != ktLetter) || keyValue == 0 {
				continue
			}
			character := rune(keyValue)
			if _, ok := keymap.keyComboMap[character]; ok {
				continue
			}
			keymap.keyComboMap[character] = KeyCombo{key, activeShiftKeys}
		}
	}
	return &keymap, nil
}
