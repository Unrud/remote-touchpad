/*
 *    Copyright (c) 2018-2019, 2023 Unrud <unrud@outlook.com>
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

export default class Mouse {
    #moveSpeed = 1;
    #scrollSpeed = 1;

    #buttons = 0;
    #inputController;

    constructor (inputController, element) {
        this.#inputController = inputController;
        for (const type of ["touchstart", "touchend", "touchcancel", "touchmove"]) {
            element.addEventListener(type, (event) => {
                event.preventDefault();
            });
        }
        element.addEventListener("mousedown", this.#handleMouseDownAndUp.bind(this));
        element.addEventListener("mouseup", this.#handleMouseDownAndUp.bind(this));
        element.addEventListener("mousemove", this.#handleMousemove.bind(this));
        element.addEventListener("wheel", this.#handleWheel.bind(this));
    }

    configure(config) {
        this.#moveSpeed = config.mouseMoveSpeed;
        this.#scrollSpeed = config.mouseScrollSpeed;
    }

    #updateButtons(newButtons) {
        for (let button = 0; button < 3; button += 1) {
            const flag = 1 << button;
            if ((newButtons&flag) != (this.#buttons & flag)) {
                this.#inputController.pointerButton(button, newButtons & flag);
            }
        }
        this.#buttons = newButtons;
    }

    #handleMouseDownAndUp(event) {
        this.#updateButtons(event.buttons);
    }

    #handleMousemove(event) {
        this.#inputController.pointerMove(
            event.movementX * this.#moveSpeed, event.movementY * this.#moveSpeed);
    }

    #handleWheel(event) {
        if (event.deltaMode == WheelEvent.DOM_DELTA_PIXEL) {
            this.#inputController.pointerScroll(
                event.deltaX * this.#scrollSpeed, event.deltaY * this.#scrollSpeed, true);
        } else if (event.deltaMode == WheelEvent.DOM_DELTA_LINE) {
            this.#inputController.pointerScroll(
                event.deltaX * 20 * this.#scrollSpeed, event.deltaY * 20 * this.#scrollSpeed,
                true);
        }
    }
}
