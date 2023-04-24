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

import {
    KEY_SUPER, KEY_BACK_SPACE, KEY_RETURN, KEY_DELETE, KEY_HOME, KEY_END,
    KEY_LEFT, KEY_RIGHT, KEY_UP, KEY_DOWN,
} from "./inputcontroller.mjs";

export default class Keyboard {
    #inputController;
    #checkAllowedCallback;

    constructor(inputController, checkAllowedCallback) {
        this.#inputController = inputController;
        this.#checkAllowedCallback = checkAllowedCallback;
        document.addEventListener("keydown", this.#handleKeydown.bind(this));
    }

    configure() {}

    #handleKeydown(event) {
        if (!this.#checkAllowedCallback() ||
            event.ctrlKey || event.altKey || event.isComposing) {
            return;
        }
        let key = null;
        if (event.key == "OS" || event.key == "Super" || event.key == "Meta") {
            key = KEY_SUPER;
        } else if (event.key == "Backspace") {
            key = KEY_BACK_SPACE;
        } else if (event.key == "Enter") {
            key = KEY_RETURN;
        } else if (event.key == "Delete") {
            key = KEY_DELETE;
        } else if (event.key == "Home") {
            key = KEY_HOME;
        } else if (event.key == "End") {
            key = KEY_END;
        } else if (event.key == "Left" || event.key == "ArrowLeft") {
            key = KEY_LEFT;
        } else if (event.key == "Right" || event.key == "ArrowRight") {
            key = KEY_RIGHT;
        } else if (event.key == "Up" || event.key == "ArrowUp") {
            key = KEY_UP;
        } else if (event.key == "Down" || event.key == "ArrowDown") {
            key = KEY_DOWN;
        }
        if (key != null) {
            if (!event.shiftKey) {
                event.preventDefault();
                this.#inputController.keyboardKey(key);
            }
        } else if (event.key.length == 1) {
            event.preventDefault();
            this.#inputController.keyboardText(event.key);
        }
    }
}
