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

var ws;

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

function updateMoveAndScroll() {
    var moveX = Math.trunc(moveXSum);
    var moveY = Math.trunc(moveYSum);
    if (Math.abs(moveX) >= 1 || Math.abs(moveY) >= 1) {
        moveXSum -= moveX;
        moveYSum -= moveY;
        ws.send("m" + moveX + ";" + moveY);
    }
    var scrollX = Math.trunc(scrollXSum);
    var scrollY = Math.trunc(scrollYSum);
    if (Math.abs(scrollX) >= 1 || Math.abs(scrollY) >= 1) {
        scrollXSum -= scrollX;
        scrollYSum -= scrollY;
        scrolling = true;
        ws.send("s" + scrollX + ";" + scrollY);
    }
}

function handleStart(evt) {
    // Might get called multiple times for the same touches
    if (ongoingTouches.length == 0) {
        touchStart = evt.timeStamp;
        touchMoved = false;
    }
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i += 1) {
        if (ongoingTouches.length == 0 && !touches[i].target.classList.contains("touchpad")) {
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
        if (!dragging) {
            moveXSum = Math.trunc(moveXSum);
            moveYSum = Math.trunc(moveYSum);
        }
        scrollXSum = Math.trunc(scrollXSum);
        scrollYSum = Math.trunc(scrollYSum);
        if (draggingTimeout != null) {
            clearTimeout(draggingTimeout);
            draggingTimeout = null;
            dragging = true;
        }
        if (scrolling) {
            ws.send("sf");
            scrolling = false;
        }
    }
}

function handleEnd(evt) {
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i += 1) {
        var idx = ongoingTouchIndexById(touches[i].identifier);
        if (idx < 0) {
            continue;
        }
        ongoingTouches.splice(idx, 1);
        touchReleasedCount += 1;
        touchLastEnd = evt.timeStamp;
        if (scrolling) {
            ws.send("sf");
            scrolling = false;
        }
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

function handleMove(evt) {
    var sumX = 0;
    var sumY = 0;
    var touches = evt.changedTouches;
    for (var i = 0; i < touches.length; i += 1) {
        var idx = ongoingTouchIndexById(touches[i].identifier);
        if (idx < 0) {
            continue;
        }
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
            moveXSum += sumX;
            moveYSum += sumY;
        } else if (ongoingTouches.length == 2) {
            scrollXSum -= sumX;
            scrollYSum -= sumY;
        }
        updateMoveAndScroll();
    }
}

function handleKeydown(evt) {
    var key = {
        "OS": KEY_SUPER,
        "Super": KEY_SUPER,
        "Meta": KEY_SUPER,
        "Backspace": KEY_BACK_SPACE,
        "Enter": KEY_RETURN,
        "Delete": KEY_DELETE,
        "Home": KEY_HOME,
        "End": KEY_END,
        "Left": KEY_LEFT,
        "ArrowLeft": KEY_LEFT,
        "Right": KEY_RIGHT,
        "ArrowRight": KEY_RIGHT,
        "Up": KEY_UP,
        "ArrowUp": KEY_UP,
        "Down": KEY_DOWN,
        "ArrowDown": KEY_DOWN
    }[evt.key];
    if (key != undefined) {
        evt.preventDefault();
        ws.send("k" + key);
    } else if (evt.key.length == 1) {
        evt.preventDefault();
        ws.send("t" + evt.key);
    }
}

function challengeResponse(message) {
    var shaObj = new jsSHA("SHA-256", "TEXT");
    shaObj.setHMACKey(message, "TEXT");
    shaObj.update(window.location.hash.substr(1));
    return btoa(shaObj.getHMAC("BYTES"));
}

window.addEventListener("load", function() {
    var authenticated = false;
    var opening = document.getElementById("opening");
    var closed = document.getElementById("closed");
    var pad = document.getElementById("pad");
    var keys = document.getElementById("keys");
    var keysPages = keys.querySelectorAll(".page");
    var keyboard = document.getElementById("keyboard");
    var fullscreenbutton = document.getElementById("fullscreenbutton");
    var keyboardtext = document.getElementById("keyboardtext");
    var activeScene;

    function showScene(scene) {
        activeScene = scene;
        if (fullscreenElement() && !scene.classList.contains("fullscreen")) {
            exitFullscreen();
        }
        keyboardtext.value = "";
        [opening, closed, pad, keys, keyboard].forEach(function(element) {
            element.classList.toggle("hidden", element != scene);
        });
    }

    function showKeys(pageIndex) {
        if (!Number.isInteger(pageIndex) || pageIndex < 0 || keysPages.length <= pageIndex) {
            pageIndex = 0;
        }
        showScene(keys);
        for (var i = 0; i < keysPages.length; i += 1) {
            keysPages[i].classList.toggle("hidden", i != pageIndex);
        }
        if ((history.state || "").split(":")[0] == "keys") {
            history.replaceState("keys:" + pageIndex, "");
        } else {
            history.pushState("keys:" + pageIndex, "");
        }
    }

    function showKeyboard() {
        showScene(keyboard);
        keyboardtext.value = sessionStorage.getItem("keyboard") || "";
        keyboardtext.focus();
        if (history.state != "keyboard") {
            history.pushState("keyboard", "");
        }
    }

    keyboardtext.oninput = function() {
        sessionStorage.setItem("keyboard", keyboardtext.value);
    };

    showScene(opening);

    var wsURL = new URL("ws", location.href);
    wsURL.protocol = wsURL.protocol == "http:" ? "ws:" : "wss:";
    ws = new WebSocket(wsURL);

    ws.onmessage = function(evt) {
        if (authenticated) {
            ws.close();
            return;
        }
        ws.send(challengeResponse(evt.data));
        authenticated = true;
        window.onpopstate();
    };

    ws.onclose = function() {
        authenticated = false;
        showScene(closed);
    };

    document.getElementById("keysbutton").addEventListener("click", function() {
        showKeys(0);
    });
    document.getElementById("keyboardbutton").addEventListener("click", function() {
        showKeyboard();
    });
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
        var pageIndex = 0;
        for (var i = 0; i < keysPages.length; i += 1) {
            if (!keysPages[i].classList.contains("hidden")) {
                pageIndex = i;
            }
        }
        showKeys(pageIndex + 1);
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
        showKeys(1);
    });
    document.getElementById("sendbutton").addEventListener("click", function() {
        if (keyboardtext.value) {
            ws.send("t" + keyboardtext.value);
            keyboardtext.value = "";
            keyboardtext.oninput();
        }
        window.history.back();
    });
    window.onpopstate = function() {
        if (authenticated) {
            if ((history.state || "").split(":")[0] == "keys") {
                showKeys(parseInt(history.state.split(":")[1]) || 0);
            } else if (history.state == "keyboard") {
                showKeyboard();
            } else {
                showScene(pad);
            }
        }
    };
    document.getElementById("reloadbutton").addEventListener("click", function() {
        location.reload();
    });
    document.addEventListener("touchstart", handleStart);
    document.addEventListener("touchend", handleEnd);
    document.addEventListener("touchcancel", handleEnd);
    document.addEventListener("touchmove", handleMove);
    window.addEventListener("keydown", function(evt) {
        if (activeScene && activeScene.classList.contains("key")) {
            handleKeydown(evt);
        }
    });
});
