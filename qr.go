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

import qrcode "github.com/skip2/go-qrcode"

func GenerateQRCode(message string, invert bool) (string, error) {
	q, err := qrcode.New(message, qrcode.Medium)
	if err != nil {
		return "", err
	}
	return qrCodeToString(q.Bitmap(), invert), nil
}

func qrCodeToString(bits [][]bool, invert bool) string {
	s := ""
	for y := -1; y < len(bits); y += 2 {
		for x := range bits[0] {
			upper := !invert
			if 0 <= y && y < len(bits) {
				upper = bits[y][x]
			}
			lower := !invert
			if 0 <= y+1 && y+1 < len(bits) {
				lower = bits[y+1][x]
			}
			if upper != invert && lower != invert {
				s += " "
			} else if upper == invert && lower != invert {
				s += "▀"
			} else if upper != invert && lower == invert {
				s += "▄"
			} else {
				s += "█"
			}
		}
		s += "\n"
	}
	return s
}
