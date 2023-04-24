//go:build null

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
	"log"
)

type nullController struct{}

func init() {
	RegisterController("null", InitNullController, 1000)
}

func InitNullController() (Controller, error) {
	return &nullController{}, nil
}

func (p *nullController) Close() error {
	return nil
}

func (p *nullController) KeyboardText(text string) error {
	log.Printf("KeyboardText(text: %#v)", text)
	return nil
}

func (p *nullController) KeyboardKey(key Key) error {
	log.Printf("KeyboardKey(key: %#v)", key)
	return nil
}

func (p *nullController) PointerButton(button PointerButton, press bool) error {
	log.Printf("PointerButton(button: %#v, press: %#v)", button, press)
	return nil
}

func (p *nullController) PointerMove(deltaX, deltaY int) error {
	log.Printf("PointerMove(deltaX: %#v, deltaY: %#v)", deltaX, deltaY)
	return nil
}

func (p *nullController) PointerScroll(deltaHorizontal, deltaVertical int, finish bool) error {
	log.Printf("PointerScroll(deltaHorizontal: %#v, deltaVertical: %#v, finish: %#v)",
		deltaHorizontal, deltaVertical, finish)
	return nil
}
