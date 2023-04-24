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

export const POINTER_BUTTON_LEFT = 0;
export const POINTER_BUTTON_RIGHT = 1;
export const POINTER_BUTTON_MIDDLE = 2;

export const KEY_VOLUME_MUTE = 0;
export const KEY_VOLUME_DOWN = 1;
export const KEY_VOLUME_UP = 2;
export const KEY_MEDIA_PLAY_PAUSE = 3;
export const KEY_MEDIA_PREV_TRACK = 4;
export const KEY_MEDIA_NEXT_TRACK = 5;
export const KEY_BROWSER_BACK = 6;
export const KEY_BROWSER_FORWARD = 7;
export const KEY_SUPER = 8;
export const KEY_LEFT = 9;
export const KEY_RIGHT = 10;
export const KEY_UP = 11;
export const KEY_DOWN = 12;
export const KEY_HOME = 13;
export const KEY_END = 14;
export const KEY_BACK_SPACE = 15;
export const KEY_DELETE = 16;
export const KEY_RETURN = 17;

export default class InputController {
    #updateRate = 0;

    #moveXSum = 0;
    #moveYSum = 0;
    #scrollHSum = 0;
    #scrollVSum = 0;
    #scrolling = false;
    #scrollFinish = false;
    #updateTimeoutActive = false;
    #socket;

    constructor(socket) {
        this.#socket = socket;
    }

    configure(config) {
        this.#updateRate = config.updateRate;
    }

    #startUpdate(fromTimeout) {
        if (this.#updateTimeoutActive && !fromTimeout) {
            return;
        }
        this.#updateTimeoutActive = false;
        let finished = true;
        const xInt = Math.trunc(this.#moveXSum);
        const yInt = Math.trunc(this.#moveYSum);
        if (xInt != 0 || yInt != 0) {
            this.#socket.send("m" + xInt + ";" + yInt);
            this.#moveXSum -= xInt;
            this.#moveYSum -= yInt;
            finished = false;
        }
        const hInt = Math.trunc(this.#scrollHSum);
        const vInt = Math.trunc(this.#scrollVSum);
        if (hInt != 0 || vInt != 0) {
            this.#socket.send((this.#scrollFinish ? "S" : "s") + hInt + ";" + vInt);
            this.#scrollHSum -= hInt;
            this.#scrollVSum -= vInt;
            this.#scrolling = !this.#scrollFinish;
            this.#scrollFinish = false;
            finished = false;
        } else if (this.#scrollFinish && this.#scrolling) {
            this.#socket.send("S");
            this.#scrolling = false;
            this.#scrollFinish = false;
        }
        this.#updateTimeoutActive = !finished && this.#updateRate > 0;
        if (this.#updateTimeoutActive) {
            setTimeout(this.#startUpdate.bind(this), 1000 / this.#updateRate, true);
        }
    }

    pointerMove(deltaX, deltaY) {
        this.#moveXSum += deltaX;
        this.#moveYSum += deltaY;
        this.#startUpdate();
    }

    pointerScroll(deltaHorizontal, deltaVertical, finish) {
        this.#scrollHSum += deltaHorizontal;
        this.#scrollVSum += deltaVertical;
        this.#scrollFinish |= finish;
        this.#startUpdate();
    };

    pointerButton(button, press) {
        this.#socket.send("b" + button + ";" + (press ? 1 : 0));
    }

    keyboardKey(key) {
        this.#socket.send("k" + key);
    }

    keyboardText(text) {
        this.#socket.send("t" + text);
    }
}
