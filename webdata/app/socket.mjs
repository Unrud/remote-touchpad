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

import jsSHA from "./sha256.mjs"

const challengeResponse = (message, secret) => {
    const shaObj = new jsSHA("SHA-256", "TEXT");
    shaObj.setHMACKey(message, "TEXT");
    shaObj.update(secret);
    return btoa(shaObj.getHMAC("BYTES"));
};

export default class Socket extends EventTarget {
    #secret;
    #authenticated;
    #ws;

    constructor(url, secret) {
        super();
        this.#secret = secret;
        this.#authenticated = false;
        this.#ws = new WebSocket(url);
        this.#ws.addEventListener("message", this.#handle_ws_message.bind(this));
        this.#ws.addEventListener("close", this.#handle_ws_close.bind(this));
    }

    #handle_ws_message(event) {
        if (!this.#authenticated) {
            this.#ws.send(challengeResponse(event.data, this.#secret));
            this.#authenticated = true;
            return;
        }
        let config;
        try {
            config = JSON.parse(event.data);
        } catch (e) {
            this.#ws.close();
            throw (e);
        }
        this.dispatchEvent(new CustomEvent("config", {detail: config}));
    }

    #handle_ws_close() {
        this.dispatchEvent(new CustomEvent("close"));
    }

    send(message) {
        this.#ws.send(message);
    }
}
