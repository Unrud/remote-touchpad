"use strict";
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

(() => {

// [1 Touch, 2 Touches, 3 Touches] (as pixel)
const TOUCH_MOVE_THRESHOLD = [10, 15, 15];
// Max time between consecutive touches for clicking or dragging (as milliseconds)
const TOUCH_TIMEOUT = 250;
// [[pixel/second, multiplicator], ...]
const POINTER_ACCELERATION = [
    [0, 0],
    [87, 1],
    [173, 1],
    [553, 2],
];

const POINTER_BUTTON_LEFT = 0;
const POINTER_BUTTON_RIGHT = 1;
const POINTER_BUTTON_MIDDLE = 2;

const KEY_VOLUME_MUTE = 0;
const KEY_VOLUME_DOWN = 1;
const KEY_VOLUME_UP = 2;
const KEY_MEDIA_PLAY_PAUSE = 3;
const KEY_MEDIA_PREV_TRACK = 4;
const KEY_MEDIA_NEXT_TRACK = 5;
const KEY_BROWSER_BACK = 6;
const KEY_BROWSER_FORWARD = 7;
const KEY_SUPER = 8;
const KEY_LEFT = 9;
const KEY_RIGHT = 10;
const KEY_UP = 11;
const KEY_DOWN = 12;
const KEY_HOME = 13;
const KEY_END = 14;
const KEY_BACK_SPACE = 15;
const KEY_DELETE = 16;
const KEY_RETURN = 17;

const wsURL = new URL("ws", location.href);
wsURL.protocol = wsURL.protocol == "http:" ? "ws:" : "wss:";
const ws = new WebSocket(wsURL);

let config = null;

const compat = (() => {
    const compat = {};

    compat.fullscreenEnabled = () => {
        return (document.fullscreenEnabled ||
            document.webkitFullscreenEnabled ||
            false);
    };

    compat.requestFullscreen = (element, options) => {
        if (element.requestFullscreen) {
            element.requestFullscreen(options);
        } else if (element.webkitRequestFullscreen) {
            element.webkitRequestFullscreen(options);
        }
    };

    compat.exitFullscreen = () => {
        if (document.exitFullscreen) {
            document.exitFullscreen();
        } else if (document.webkitExitFullscreen) {
            document.webkitExitFullscreen();
        }
    };

    compat.fullscreenElement = () => {
        return (document.fullscreenElement ||
            document.webkitFullscreenElement ||
            null);
    };

    compat.addFullscreenchangeEventListener = (listener) => {
        if ("onfullscreenchange" in document) {
            document.addEventListener("fullscreenchange", listener);
        } else if ("onwebkitfullscreenchange" in document) {
            document.addEventListener("webkitfullscreenchange", listener);
        }
    };

    compat.requestPointerLock = (element) => {
        if (element.requestPointerLock) {
            element.requestPointerLock();
        }
    };

    compat.exitPointerLock = () => {
        if (document.exitPointerLock) {
            document.exitPointerLock();
        }
    };

    compat.pointerLockElement = () => {
        return document.pointerLockElement || null;
    };

    compat.addPointerlockchangeEventListener = (listener) => {
        if ("onpointerlockchange" in document) {
            document.addEventListener("pointerlockchange", listener);
        }
    };

    return compat;
})();

const controller = (() => {
    const controller = {};

    let moveXSum = 0;
    let moveYSum = 0;
    let scrollHSum = 0;
    let scrollVSum = 0;
    let scrolling = false;
    let scrollFinish = false;
    let updateTimeoutActive = false;

    const startUpdate = (fromTimeout) => {
        if (updateTimeoutActive && !fromTimeout) {
            return;
        }
        updateTimeoutActive = false;
        let finished = true;
        const xInt = Math.trunc(moveXSum);
        const yInt = Math.trunc(moveYSum);
        if (xInt != 0 || yInt != 0) {
            ws.send("m" + xInt + ";" + yInt);
            moveXSum -= xInt;
            moveYSum -= yInt;
            finished = false;
        }
        const hInt = Math.trunc(scrollHSum);
        const vInt = Math.trunc(scrollVSum);
        if (hInt != 0 || vInt != 0) {
            ws.send((scrollFinish ? "S" : "s") + hInt + ";" + vInt);
            scrollHSum -= hInt;
            scrollVSum -= vInt;
            scrolling = !scrollFinish;
            scrollFinish = false;
            finished = false;
        } else if (scrollFinish && scrolling) {
            ws.send("S");
            scrolling = false;
            scrollFinish = false;
        }
        updateTimeoutActive = !finished && config.updateRate > 0;
        if (updateTimeoutActive) {
            setTimeout(startUpdate, 1000/config.updateRate, true);
        }
    };

    controller.pointerMove = (deltaX, deltaY) => {
        moveXSum += deltaX;
        moveYSum += deltaY;
        startUpdate();
    };

    controller.pointerScroll = (deltaHorizontal, deltaVertical, finish) => {
        scrollHSum += deltaHorizontal;
        scrollVSum += deltaVertical;
        scrollFinish |= finish;
        startUpdate();
    };

    controller.pointerButton = (button, press) => {
        ws.send("b" + button + ";" + (press ? 1 : 0));
    };


    controller.keyboardKey = (key) => {
        ws.send("k" + key);
    };

    controller.keyboardText = (text) => {
        ws.send("t" + text);
    };

    return controller;
})();

const touchpad = (() => {
    const touchpad = {};

    let moved = false;
    let startTimeStamp = 0;
    let lastEndTimeStamp = 0;
    let releasedCount = 0;
    let ongoingTouches = [];
    let dragging = false;
    let draggingTimeout = null;

    const copyTouch = (touch, timeStamp) => {
        return {
            identifier: touch.identifier,
            pageX: touch.pageX,
            pageXStart: touch.pageX,
            pageY: touch.pageY,
            pageYStart: touch.pageY,
            timeStamp: timeStamp,
        };
    };

    const ongoingTouchIndexById = (idToFind) => {
        for (let i = 0; i < ongoingTouches.length; i += 1) {
            if (ongoingTouches[i].identifier == idToFind) {
                return i;
            }
        }
        return -1;
    };

    const calculateAccelerationMult = (speed) => {
        for (let i = 0; i < POINTER_ACCELERATION.length; i += 1) {
            const s2 = POINTER_ACCELERATION[i][0];
            const a2 = POINTER_ACCELERATION[i][1];
            if (s2 <= speed) {
                continue;
            }
            if (i == 0) {
                return a2;
            }
            const s1 = POINTER_ACCELERATION[i - 1][0];
            const a1 = POINTER_ACCELERATION[i - 1][1];
            return ((speed - s1) / (s2 - s1)) * (a2 - a1) + a1;
        }
        if (POINTER_ACCELERATION.length > 0) {
            return POINTER_ACCELERATION[POINTER_ACCELERATION.length - 1][1];
        }
        return 1;
    };

    const onDraggingTimeout = () => {
        draggingTimeout = null;
        controller.pointerButton(POINTER_BUTTON_LEFT, false);
    };

    touchpad.handleTouchstart = (evt) => {
        // Might get called multiple times for the same touches
        if (ongoingTouches.length == 0) {
            startTimeStamp = evt.timeStamp;
            moved = false;
        }
        const touches = evt.changedTouches;
        let foundTouch = false;
        for (let i = 0; i < touches.length; i += 1) {
            if (ongoingTouches.length == 0 && !touches[i].target.classList.contains("touch-input")) {
                continue;
            }
            foundTouch = true;
            const touch = copyTouch(touches[i], evt.timeStamp);
            const idx = ongoingTouchIndexById(touch.identifier);
            if (idx < 0) {
                ongoingTouches.push(touch);
            } else {
                ongoingTouches[idx] = touch;
            }
        }
        if (!foundTouch) {
            return;
        }
        evt.preventDefault();
        lastEndTimeStamp = 0;
        if (draggingTimeout != null) {
            clearTimeout(draggingTimeout);
            draggingTimeout = null;
            dragging = true;
        }
        controller.pointerScroll(0, 0, true);
    };

    touchpad.handleTouchend = touchpad.handleTouchcancel = (evt) => {
        const touches = evt.changedTouches;
        let foundTouch = false;
        for (let i = 0; i < touches.length; i += 1) {
            const idx = ongoingTouchIndexById(touches[i].identifier);
            if (idx < 0) {
                continue;
            }
            foundTouch = true;
            ongoingTouches.splice(idx, 1);
            releasedCount += 1;
        }
        if (!foundTouch) {
            return;
        }
        evt.preventDefault();
        lastEndTimeStamp = evt.timeStamp;
        controller.pointerScroll(0, 0, true);
        if (releasedCount > TOUCH_MOVE_THRESHOLD.length) {
            moved = true;
        }
        if (ongoingTouches.length == 0 && releasedCount >= 1) {
            if (dragging) {
                dragging = false;
                controller.pointerButton(POINTER_BUTTON_LEFT, false);
            }
            if (!moved && evt.timeStamp - startTimeStamp < TOUCH_TIMEOUT) {
                let button = 0;
                if (releasedCount == 1) {
                    button = POINTER_BUTTON_LEFT;
                } else if (releasedCount == 2) {
                    button = POINTER_BUTTON_RIGHT;
                } else if (releasedCount == 3) {
                    button = POINTER_BUTTON_MIDDLE;
                }
                controller.pointerButton(button, true);
                if (button == POINTER_BUTTON_LEFT) {
                    draggingTimeout = setTimeout(onDraggingTimeout, TOUCH_TIMEOUT);
                } else {
                    controller.pointerButton(button, false);
                }
            }
            releasedCount = 0;
        }
    };

    touchpad.handleTouchmove = (evt) => {
        let sumX = 0;
        let sumY = 0;
        const touches = evt.changedTouches;
        let foundTouch = false;
        for (let i = 0; i < touches.length; i += 1) {
            const idx = ongoingTouchIndexById(touches[i].identifier);
            if (idx < 0) {
                continue;
            }
            foundTouch = true;
            if (!moved) {
                const dist = Math.sqrt(Math.pow(touches[i].pageX - ongoingTouches[idx].pageXStart, 2) +
                    Math.pow(touches[i].pageY - ongoingTouches[idx].pageYStart, 2));
                if (ongoingTouches.length > TOUCH_MOVE_THRESHOLD.length ||
                    dist > TOUCH_MOVE_THRESHOLD[ongoingTouches.length - 1] ||
                    evt.timeStamp - startTimeStamp >= TOUCH_TIMEOUT) {
                    moved = true;
                }
            }
            const dx = touches[i].pageX - ongoingTouches[idx].pageX;
            const dy = touches[i].pageY - ongoingTouches[idx].pageY;
            const timeDelta = evt.timeStamp - ongoingTouches[idx].timeStamp;
            sumX += dx * calculateAccelerationMult(Math.abs(dx) / timeDelta * 1000);
            sumY += dy * calculateAccelerationMult(Math.abs(dy) / timeDelta * 1000);
            ongoingTouches[idx].pageX = touches[i].pageX;
            ongoingTouches[idx].pageY = touches[i].pageY;
            ongoingTouches[idx].timeStamp = evt.timeStamp;
        }
        if (!foundTouch) {
            return;
        }
        evt.preventDefault();
        if (moved && evt.timeStamp - lastEndTimeStamp >= TOUCH_TIMEOUT) {
            if (ongoingTouches.length == 1 || dragging) {
                controller.pointerMove(sumX*config.moveSpeed, sumY*config.moveSpeed);
            } else if (ongoingTouches.length == 2) {
                controller.pointerScroll(-sumX*config.scrollSpeed, -sumY*config.scrollSpeed, false);
            }
        }
    };

    return touchpad;
})();

const keyboard = (() => {
    const keyboard = {};

    keyboard.handleKeydown = (evt) => {
        if (evt.ctrlKey || evt.altKey || evt.isComposing) {
            return;
        }
        let key = null;
        if (evt.key == "OS" || evt.key == "Super" || evt.key == "Meta") {
            key = KEY_SUPER;
        } else if (evt.key == "Backspace") {
            key = KEY_BACK_SPACE;
        } else if (evt.key == "Enter") {
            key = KEY_RETURN;
        } else if (evt.key == "Delete") {
            key = KEY_DELETE;
        } else if (evt.key == "Home") {
            key = KEY_HOME;
        } else if (evt.key == "End") {
            key = KEY_END;
        } else if (evt.key == "Left" || evt.key == "ArrowLeft") {
            key = KEY_LEFT;
        } else if (evt.key == "Right" || evt.key == "ArrowRight") {
            key = KEY_RIGHT;
        } else if (evt.key == "Up" || evt.key == "ArrowUp") {
            key = KEY_UP;
        } else if (evt.key == "Down" || evt.key == "ArrowDown") {
            key = KEY_DOWN;
        }
        if (key != null) {
            if (!evt.shiftKey) {
                evt.preventDefault();
                controller.keyboardKey(key);
            }
        } else if (evt.key.length == 1) {
            evt.preventDefault();
            controller.keyboardText(evt.key);
        }
    };

    return keyboard;
})();

const mouse = (() => {
    const mouse = {};

    let buttons = 0;

    const updateButtons = (newButtons) => {
        for (let button = 0; button < 3; button += 1) {
            const flag = 1 << button;
            if ((newButtons&flag) != (buttons&flag)) {
                controller.pointerButton(button, newButtons&flag);
            }
        }
        buttons = newButtons;
    };

    mouse.handleMousedown = mouse.handleMouseup = (evt) => {
        updateButtons(evt.buttons);
    };

    mouse.handleMousemove = (evt) => {
        controller.pointerMove(evt.movementX*config.mouseMoveSpeed, evt.movementY*config.mouseMoveSpeed);
    };

    mouse.handleWheel = (evt) => {
        if (evt.deltaMode == WheelEvent.DOM_DELTA_PIXEL) {
            controller.pointerScroll(evt.deltaX*config.mouseScrollSpeed, evt.deltaY*config.mouseScrollSpeed, true);
        } else if (evt.deltaMode == WheelEvent.DOM_DELTA_LINE) {
            controller.pointerScroll(evt.deltaX*20*config.mouseScrollSpeed, evt.deltaY*20*config.mouseScrollSpeed, true);
        }
    };

    return mouse;
})();

const challengeResponse = (message) => {
    const shaObj = new window.jsSHA("SHA-256", "TEXT");
    shaObj.setHMACKey(message, "TEXT");
    shaObj.update(window.location.hash.substr(1));
    return btoa(shaObj.getHMAC("BYTES"));
};

const scenes = document.querySelectorAll("body > .scene");
const openingScene = document.getElementById("opening");
const closedScene = document.getElementById("closed");
const padScene = document.getElementById("pad");
const keysScene = document.getElementById("keys");
const keysPages = keysScene.querySelectorAll(":scope > .page");
const textInputScene = document.getElementById("text-input");
const textInput = textInputScene.querySelector("textarea");
const mouseScene = document.getElementById("mouse");

let ready = false;
let closed = false;
let activeScene = null;
let keysActiveName = "";

const showScene = (scene) => {
    activeScene = scene;
    if (compat.fullscreenElement() && !scene.classList.contains("allow-fullscreen")) {
        compat.exitFullscreen();
    }
    if (compat.pointerLockElement() && activeScene != mouseScene) {
        compat.exitPointerLock();
    }
    textInput.value = "";
    for (const otherScene of scenes) {
        otherScene.classList.toggle("hidden", otherScene != scene);
    }
};

const setKeysPage = (index, relative=false) => {
    if (relative) {
        for (let i = 0; i < keysPages.length && keysPages[i].classList.contains("hidden"); i += 1, index += 1);
    }
    index = ((index % keysPages.length) + keysPages.length) % keysPages.length;
    sessionStorage.setItem(keysActiveName, index);
    for (let i = 0; i < keysPages.length; i += 1) {
        keysPages[i].classList.toggle("hidden", i != index);
    }
};

const showKeys = (name = "", defaultIndex = 0) => {
    showScene(keysScene);
    keysActiveName = "keys" + (name ? ":" + name : "");
    let keysIndex = parseInt(sessionStorage.getItem(keysActiveName));
    if (isNaN(keysIndex)) {
        keysIndex = defaultIndex;
    }
    setKeysPage(keysIndex);
    if (history.state != keysActiveName) {
        history.pushState(keysActiveName, "");
    }
};

const showTextInput = () => {
    showScene(textInputScene);
    textInput.value = sessionStorage.getItem("text-input") || "";
    textInput.focus();
    if (history.state != "text-input") {
        history.pushState("text-input", "");
    }
};

textInput.oninput = () => {
    sessionStorage.setItem("text-input", textInput.value);
};

const updateUI = () => {
    if (!ready) {
        showScene(closed ? closedScene : openingScene);
    } else if (compat.pointerLockElement()) {
        showScene(mouseScene);
    } else if ((history.state || "").split(":")[0] == "keys") {
        showKeys(history.state.substr("keys:".length));
    } else if (history.state == "text-input") {
        showTextInput();
    } else {
        showScene(padScene);
    }
};

let authenticated = false;
ws.onmessage = (evt) => {
    if (!authenticated) {
        ws.send(challengeResponse(evt.data));
        authenticated = true;
        return;
    }
    try {
        config = JSON.parse(evt.data);
    } catch (e) {
        ws.close();
        throw (e);
    }
    ready = true;
    updateUI();
};

ws.onclose = () => {
    ready = false;
    closed = true;
    updateUI();
};

compat.addFullscreenchangeEventListener(() => {
    updateUI();
});
for (const element of document.querySelectorAll(".visble-if-fullscreen-enabled")) {
    element.classList.toggle("hidden", !compat.fullscreenEnabled());
}

const toggleFullscreen = () => {
    if (compat.fullscreenElement()) {
        compat.exitFullscreen();
    } else {
        compat.requestFullscreen(document.documentElement, {navigationUI: "hide"});
    }
};

document.getElementById("send-text").addEventListener("click", () => {
    if (textInput.value) {
        // normalize line endings
        controller.keyboardText(textInput.value.replace(/\r\n?/g, "\n"));
        textInput.value = "";
        textInput.oninput();
    }
    history.back();
});
window.addEventListener("popstate", () => {
    updateUI();
});
document.addEventListener("touchstart", touchpad.handleTouchstart);
document.addEventListener("touchend", touchpad.handleTouchend);
document.addEventListener("touchcancel", touchpad.handleTouchcancel);
document.addEventListener("touchmove", touchpad.handleTouchmove);
document.addEventListener("keydown", (evt) => {
    if (activeScene && activeScene.classList.contains("keyboard-input")) {
        keyboard.handleKeydown(evt);
    }
});
compat.addPointerlockchangeEventListener(() => {
    updateUI();
});
document.addEventListener("mousedown", (event) => {
    if (activeScene != mouseScene && event.buttons == 1 && event.target.classList.contains("mouse-input")) {
        compat.requestPointerLock(mouseScene);
    }
});
for (const type of ["touchstart", "touchend", "touchcancel", "touchmove"]) {
    mouseScene.addEventListener(type, (evt) => {
        evt.preventDefault();
    });
}
mouseScene.addEventListener("mousedown", mouse.handleMousedown);
mouseScene.addEventListener("mouseup", mouse.handleMouseup);
mouseScene.addEventListener("mousemove", mouse.handleMousemove);
mouseScene.addEventListener("wheel", mouse.handleWheel);

window.app = {
    KEY_VOLUME_MUTE, KEY_VOLUME_DOWN, KEY_VOLUME_UP, KEY_MEDIA_PLAY_PAUSE,
    KEY_MEDIA_PREV_TRACK, KEY_MEDIA_NEXT_TRACK, KEY_BROWSER_BACK, KEY_BROWSER_FORWARD,
    KEY_SUPER, KEY_LEFT, KEY_RIGHT, KEY_UP, KEY_DOWN, KEY_HOME, KEY_END, KEY_BACK_SPACE,
    KEY_DELETE, KEY_RETURN,
    showKeys, showTextInput, setKeysPage, toggleFullscreen,
    key: controller.keyboardKey,
    text: controller.keyboardText,
};
updateUI();

})();
