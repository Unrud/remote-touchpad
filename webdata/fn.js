"use strict";
/*
 *    Copyright (c) 2018-2019 Unrud <unrud@outlook.com>
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

// [1 Touch, 2 Touches, 3 Touches] (as pixel)
var TOUCH_MOVE_THRESHOLD = [10, 15, 15];
// Max time between consecutive touches for clicking or dragging (as milliseconds)
var TOUCH_TIMEOUT = 250;
// [[pixel/second, multiplicator], ...]
var POINTER_ACCELERATION = [
    [0, 0],
    [87, 1],
    [173, 1],
    [553, 2]
];

var POINTER_BUTTON_LEFT = 0;
var POINTER_BUTTON_RIGHT = 1;
var POINTER_BUTTON_MIDDLE = 2;

var KEY_VOLUME_MUTE = 0;
var KEY_VOLUME_DOWN = 1;
var KEY_VOLUME_UP = 2;
var KEY_MEDIA_PLAY_PAUSE = 3;
var KEY_MEDIA_PREV_TRACK = 4;
var KEY_MEDIA_NEXT_TRACK = 5;
var KEY_BROWSER_BACK = 6;
var KEY_BROWSER_FORWARD = 7;
var KEY_SUPER = 8;
var KEY_LEFT = 9;
var KEY_RIGHT = 10;
var KEY_UP = 11;
var KEY_DOWN = 12;
var KEY_HOME = 13;
var KEY_END = 14;
var KEY_BACK_SPACE = 15;
var KEY_DELETE = 16;
var KEY_RETURN = 17;

var ws = null;
var config = null;

var touchMoved = false;
var touchStart = 0;
var touchLastEnd = 0;
var touchReleasedCount = 0;
var ongoingTouches = [];
var moveXSum = 0;
var moveYSum = 0;
var scrollXSum = 0;
var scrollYSum = 0;
var dragging = false;
var draggingTimeout = null;
var scrolling = false;
var mouseButtons = 0;

function fullscreenEnabled() {
    return (document.fullscreenEnabled ||
        document.webkitFullscreenEnabled ||
        document.mozFullScreenEnabled ||
        document.msFullscreenEnabled ||
        false);
}

function requestFullscreen(element, options) {
    if (element.requestFullscreen) {
        element.requestFullscreen(options);
    } else if (element.webkitRequestFullscreen) {
        element.webkitRequestFullscreen(options);
    } else if (element.mozRequestFullScreen) {
        element.mozRequestFullScreen(options);
    } else if (element.msRequestFullscreen) {
        element.msRequestFullscreen(options);
    }
}

function exitFullscreen() {
    if (document.exitFullscreen) {
        document.exitFullscreen();
    } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen();
    } else if (document.mozCancelFullScreen) {
        document.mozCancelFullScreen();
    } else if (document.msExitFullscreen) {
        document.msExitFullscreen();
    }
}

function fullscreenElement() {
    return (document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement ||
        null);
}

function addFullscreenchangeEventListener(listener) {
    if ("onfullscreenchange" in document) {
        document.addEventListener("fullscreenchange", listener);
    } else if ("onwebkitfullscreenchange" in document) {
        document.addEventListener("webkitfullscreenchange", listener);
    }
}

function requestPointerLock(element) {
    if (element.requestPointerLock) {
        element.requestPointerLock();
    } else if (element.mozRequestPointerLock) {
        element.mozRequestPointerLock();
    }
}

function exitPointerLock() {
    if (document.exitPointerLock) {
        document.exitPointerLock();
    } else if (document.mozExitPointerLock) {
        document.mozExitPointerLock();
    }
}

function pointerLockElement() {
    return (document.pointerLockElement ||
        document.mozPointerLockElement ||
        null);
}

function addPointerlockchangeEventListener(listener) {
    if ("onpointerlockchange" in document) {
        document.addEventListener("pointerlockchange", listener);
    } else if ("onmozpointerlockchange" in document) {
        document.addEventListener("mozpointerlockchange", listener);
    }
}

function copyTouch(touch, timeStamp) {
    return {
        identifier: touch.identifier,
        pageX: touch.pageX,
        pageXStart: touch.pageX,
        pageY: touch.pageY,
        pageYStart: touch.pageY,
        timeStamp: timeStamp
    };
}

function ongoingTouchIndexById(idToFind) {
    for (var i = 0; i < ongoingTouches.length; i += 1) {
        if (ongoingTouches[i].identifier == idToFind) {
            return i;
        }
    }
    return -1;
}

function calculatePointerAccelerationMult(speed) {
    for (var i = 0; i < POINTER_ACCELERATION.length; i += 1) {
        var s2 = POINTER_ACCELERATION[i][0];
        var a2 = POINTER_ACCELERATION[i][1];
        if (s2 <= speed) {
            continue;
        }
        if (i == 0) {
            return a2;
        }
        var s1 = POINTER_ACCELERATION[i - 1][0];
        var a1 = POINTER_ACCELERATION[i - 1][1];
        return ((speed - s1) / (s2 - s1)) * (a2 - a1) + a1;
    }
    if (POINTER_ACCELERATION.length > 0) {
        return POINTER_ACCELERATION[POINTER_ACCELERATION.length - 1][1];
    }
    return 1;
}

function onDraggingTimeout() {
    draggingTimeout = null;
    ws.send("b" + POINTER_BUTTON_LEFT + ";0");
}

function updateMove(x, y) {
    moveXSum += x;
    moveYSum += y;
    var xInt = Math.trunc(moveXSum);
    var yInt = Math.trunc(moveYSum);
    if (xInt != 0 || yInt != 0) {
        moveXSum -= xInt;
        moveYSum -= yInt;
        ws.send("m" + xInt + ";" + yInt);
    }
}

function updateScroll(x, y, scrollFinish) {
    scrollXSum += x;
    scrollYSum += y;
    var xInt = Math.trunc(scrollXSum);
    var yInt = Math.trunc(scrollYSum);
    if (xInt != 0 || yInt != 0) {
        scrollXSum -= xInt;
        scrollYSum -= yInt;
        ws.send((scrollFinish ? "S" : "s") + xInt + ";" + yInt);
        scrolling = !scrollFinish;
    } else if (scrollFinish && scrolling) {
        ws.send("S");
        scrolling = false;
    }
}

function handleTouchstart(evt) {
    // Might get called multiple times for the same touches
    if (ongoingTouches.length == 0) {
        touchStart = evt.timeStamp;
        touchMoved = false;
    }
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i += 1) {
        if (ongoingTouches.length == 0 && !touches[i].target.classList.contains("touch")) {
            continue;
        }
        evt.preventDefault();
        var touch = copyTouch(touches[i], evt.timeStamp);
        var idx = ongoingTouchIndexById(touch.identifier);
        if (idx < 0) {
            ongoingTouches.push(touch);
        } else {
            ongoingTouches[idx] = touch;
        }
        touchLastEnd = 0;
        if (draggingTimeout != null) {
            clearTimeout(draggingTimeout);
            draggingTimeout = null;
            dragging = true;
        }
        updateScroll(0, 0, true);
    }
}

function handleTouchend(evt) {
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i += 1) {
        var idx = ongoingTouchIndexById(touches[i].identifier);
        if (idx < 0) {
            continue;
        }
        evt.preventDefault();
        ongoingTouches.splice(idx, 1);
        touchReleasedCount += 1;
        touchLastEnd = evt.timeStamp;
        updateScroll(0, 0, true);
    }
    if (touchReleasedCount > TOUCH_MOVE_THRESHOLD.length) {
        touchMoved = true;
    }
    if (ongoingTouches.length == 0 && touchReleasedCount >= 1) {
        if (dragging) {
            dragging = false;
            ws.send("b" + POINTER_BUTTON_LEFT + ";0");
        }
        if (!touchMoved && evt.timeStamp - touchStart < TOUCH_TIMEOUT) {
            var button = 0;
            if (touchReleasedCount == 1) {
                button = POINTER_BUTTON_LEFT;
            } else if (touchReleasedCount == 2) {
                button = POINTER_BUTTON_RIGHT;
            } else if (touchReleasedCount == 3) {
                button = POINTER_BUTTON_MIDDLE;
            }
            ws.send("b" + button + ";1");
            if (button == POINTER_BUTTON_LEFT) {
                draggingTimeout = setTimeout(onDraggingTimeout, TOUCH_TIMEOUT);
            } else {
                ws.send("b" + button + ";0");
            }
        }
        touchReleasedCount = 0;
    }
}

function handleTouchmove(evt) {
    var sumX = 0;
    var sumY = 0;
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i += 1) {
        var idx = ongoingTouchIndexById(touches[i].identifier);
        if (idx < 0) {
            continue;
        }
        evt.preventDefault();
        if (!touchMoved) {
            var dist = Math.sqrt(Math.pow(touches[i].pageX - ongoingTouches[idx].pageXStart, 2) +
                Math.pow(touches[i].pageY - ongoingTouches[idx].pageYStart, 2));
            if (ongoingTouches.length > TOUCH_MOVE_THRESHOLD.length ||
                dist > TOUCH_MOVE_THRESHOLD[ongoingTouches.length - 1] ||
                evt.timeStamp - touchStart >= TOUCH_TIMEOUT) {
                touchMoved = true;
            }
        }
        var dx = touches[i].pageX - ongoingTouches[idx].pageX;
        var dy = touches[i].pageY - ongoingTouches[idx].pageY;
        var timeDelta = evt.timeStamp - ongoingTouches[idx].timeStamp;
        sumX += dx * calculatePointerAccelerationMult(Math.abs(dx) / timeDelta * 1000);
        sumY += dy * calculatePointerAccelerationMult(Math.abs(dy) / timeDelta * 1000);
        ongoingTouches[idx].pageX = touches[i].pageX;
        ongoingTouches[idx].pageY = touches[i].pageY;
        ongoingTouches[idx].timeStamp = evt.timeStamp;
    }
    if (touchMoved && evt.timeStamp - touchLastEnd >= TOUCH_TIMEOUT) {
        if (ongoingTouches.length == 1 || dragging) {
            updateMove(sumX*config.moveSpeed, sumY*config.moveSpeed);
        } else if (ongoingTouches.length == 2) {
            updateScroll(-sumX*config.scrollSpeed, -sumY*config.scrollSpeed, false);
        }
    }
}

function handleKeydown(evt) {
    if (evt.ctrlKey || evt.altKey || evt.isComposing) {
        return;
    }
    var key = null;
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
            ws.send("k" + key);
        }
    } else if (evt.key.length == 1) {
        evt.preventDefault();
        ws.send("t" + evt.key);
    }
}

function updateMouseButtons(buttons) {
    for (var button = 0; button < 3; button += 1) {
        var flag = 1 << button;
        if ((buttons&flag) != (mouseButtons&flag)) {
            ws.send("b" + button + ";" + (buttons&flag ? 1 : 0));
        }
    }
    mouseButtons = buttons;
}

function handleMousedown(evt) {
    updateMouseButtons(evt.buttons);
}

function handleMouseup(evt) {
    updateMouseButtons(evt.buttons);
}

function handleMousemove(evt) {
    updateMove(evt.movementX*config.mouseMoveSpeed, evt.movementY*config.mouseMoveSpeed);
}

function handleWheel(evt) {
    if (evt.deltaMode == WheelEvent.DOM_DELTA_PIXEL) {
        updateScroll(evt.deltaX*config.mouseScrollSpeed, evt.deltaY*config.mouseScrollSpeed, true);
    } else if (evt.deltaMode == WheelEvent.DOM_DELTA_LINE) {
        updateScroll(evt.deltaX*20*config.mouseScrollSpeed, evt.deltaY*20*config.mouseScrollSpeed, true);
    }
}

function challengeResponse(message) {
    var shaObj = new jsSHA("SHA-256", "TEXT");
    shaObj.setHMACKey(message, "TEXT");
    shaObj.update(window.location.hash.substr(1));
    return btoa(shaObj.getHMAC("BYTES"));
}

window.addEventListener("load", function() {
    var DEFAULT_KEYS_SCENE = {
        "": 0,
        "keyboard": 1
    };
    var ready = false;
    var closed = false;
    var scenes = document.querySelectorAll("body > .scene");
    var openingScene = document.getElementById("opening");
    var closedScene = document.getElementById("closed");
    var padScene = document.getElementById("pad");
    var keysScene = document.getElementById("keys");
    var keysSubScenes = keysScene.querySelectorAll(".scene");
    var keyboardScene = document.getElementById("keyboard");
    var fullscreenbutton = document.getElementById("fullscreenbutton");
    var keyboardTextarea = keyboardScene.querySelector("textarea");
    var mouseScene = document.getElementById("mouse")
    var activeScene;
    var keysActiveName;

    function showScene(scene) {
        activeScene = scene;
        if (fullscreenElement() && !scene.classList.contains("fullscreen")) {
            exitFullscreen();
        }
        if (pointerLockElement() && activeScene != mouseScene) {
            exitPointerLock();
        }
        keyboardTextarea.value = "";
        scenes.forEach(function(otherScene) {
            otherScene.classList.toggle("hidden", otherScene != scene);
        });
    }

    function showKeysScene(index) {
        if (!Number.isInteger(index) || index < 0 || keysSubScenes.length <= index) {
            index = 0;
        }
        sessionStorage.setItem(keysActiveName, index);
        for (var i = 0; i < keysSubScenes.length; i += 1) {
            keysSubScenes[i].classList.toggle("hidden", i != index);
        }
    }

    function showKeys(name) {
        showScene(keysScene);
        keysActiveName = "keys" + (name ? ":" + name : "");
        var keysIndex = parseInt(sessionStorage.getItem(keysActiveName));
        if (isNaN(keysIndex)) {
            keysIndex = DEFAULT_KEYS_SCENE[name || ""] || 0;
        }
        showKeysScene(keysIndex);
        if (history.state != keysActiveName) {
            history.pushState(keysActiveName, "");
        }
    }

    function showKeyboard() {
        showScene(keyboardScene);
        keyboardTextarea.value = sessionStorage.getItem("keyboard") || "";
        keyboardTextarea.focus();
        if (history.state != "keyboard") {
            history.pushState("keyboard", "");
        }
    }

    keyboardTextarea.oninput = function() {
        sessionStorage.setItem("keyboard", keyboardTextarea.value);
    };

    function updateUI() {
        if (!ready) {
            showScene(closed ? closedScene : openingScene);
        } else if (pointerLockElement()) {
            showScene(mouseScene);
        } else if ((history.state || "").split(":")[0] == "keys") {
            showKeys(history.state.substr("keys:".length));
        } else if (history.state == "keyboard") {
            showKeyboard();
        } else {
            showScene(padScene);
        }
    }

    updateUI();

    var wsURL = new URL("ws", location.href);
    wsURL.protocol = wsURL.protocol == "http:" ? "ws:" : "wss:";
    ws = new WebSocket(wsURL);

    var authenticated = false;
    ws.onmessage = function(evt) {
        if (!authenticated) {
            ws.send(challengeResponse(evt.data));
            authenticated = true;
            return;
        }
        try {
            config = JSON.parse(evt.data);
        } catch (e) {
            console.log(e);
            ws.close();
            return;
        }
        ready = true;
        updateUI();
    };

    ws.onclose = function() {
        ready = false;
        closed = true;
        updateUI();
    };

    document.getElementById("keysbutton").addEventListener("click", function() {
        showKeys();
    });
    document.getElementById("keyboardbutton").addEventListener("click", function() {
        showKeyboard();
    });
    addFullscreenchangeEventListener(updateUI);
    if (!fullscreenEnabled()) {
        fullscreenbutton.classList.add("hidden");
    }
    fullscreenbutton.addEventListener("click", function() {
        if (fullscreenElement()) {
            exitFullscreen();
        } else {
            requestFullscreen(document.documentElement, {navigationUI: "hide"});
        }
    });
    document.getElementById("switchbutton").addEventListener("click", function() {
        var keysIndex = 0;
        for (var i = 0; i < keysSubScenes.length; i += 1) {
            if (!keysSubScenes[i].classList.contains("hidden")) {
                keysIndex = i;
            }
        }
        showKeysScene(keysIndex + 1);
    });
    [
        {id: "browserbackbutton", key: KEY_BROWSER_BACK},
        {id: "superbutton", key: KEY_SUPER},
        {id: "browserforwardbutton", key: KEY_BROWSER_FORWARD},
        {id: "prevtrackbutton", key: KEY_MEDIA_PREV_TRACK},
        {id: "playpausebutton", key: KEY_MEDIA_PLAY_PAUSE},
        {id: "nexttrackbutton", key: KEY_MEDIA_NEXT_TRACK},
        {id: "volumedownbutton", key: KEY_VOLUME_DOWN},
        {id: "volumemutebutton", key: KEY_VOLUME_MUTE},
        {id: "volumeupbutton", key: KEY_VOLUME_UP},
        {id: "backspacebutton", key: KEY_BACK_SPACE},
        {id: "returnbutton", key: KEY_RETURN},
        {id: "deletebutton", key: KEY_DELETE},
        {id: "homebutton", key: KEY_HOME},
        {id: "endbutton", key: KEY_END},
        {id: "leftbutton", key: KEY_LEFT},
        {id: "rightbutton", key: KEY_RIGHT},
        {id: "upbutton", key: KEY_UP},
        {id: "downbutton", key: KEY_DOWN}
    ].forEach(function(o) {
        document.getElementById(o.id).addEventListener("click", function() {
            ws.send("k" + o.key);
        });
    });
    document.getElementById("keyboardkeysbutton").addEventListener("click", function() {
        showKeys("keyboard");
    });
    document.getElementById("sendbutton").addEventListener("click", function() {
        if (keyboardTextarea.value) {
            // normalize line endings
            ws.send("t" + keyboardTextarea.value.replace(/\r\n?/g, "\n"));
            keyboardTextarea.value = "";
            keyboardTextarea.oninput();
        }
        history.back();
    });
    window.addEventListener("popstate", updateUI);
    document.getElementById("reloadbutton").addEventListener("click", function() {
        location.reload();
    });
    document.querySelectorAll(".backbutton").forEach(function(button) {
        button.addEventListener("click", function() {
            history.back();
        });
    });
    document.addEventListener("touchstart", handleTouchstart);
    document.addEventListener("touchend", handleTouchend);
    document.addEventListener("touchcancel", handleTouchend);
    document.addEventListener("touchmove", handleTouchmove);
    document.addEventListener("keydown", function(evt) {
        if (activeScene && activeScene.classList.contains("key")) {
            handleKeydown(evt);
        }
    });
    addPointerlockchangeEventListener(updateUI);
    document.addEventListener("mousedown", function(event) {
        if (activeScene != mouseScene && event.buttons == 1 && event.target.classList.contains("touch")) {
            requestPointerLock(mouseScene);
        }
    });
    ["touchstart", "touchend", "touchcancel", "touchmove"].forEach(function(type) {
        mouseScene.addEventListener(type, function(evt) {
            evt.preventDefault();
        });
    });
    mouseScene.addEventListener("mousedown", handleMousedown);
    mouseScene.addEventListener("mouseup", handleMouseup);
    mouseScene.addEventListener("mousemove", handleMousemove);
    mouseScene.addEventListener("wheel", handleWheel);
});
