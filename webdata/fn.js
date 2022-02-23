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

var util = (function() {
    var util = {};

    util.fullscreenEnabled = function() {
        return (document.fullscreenEnabled ||
            document.webkitFullscreenEnabled ||
            document.mozFullScreenEnabled ||
            document.msFullscreenEnabled ||
            false);
    };

    util.requestFullscreen = function(element, options) {
        if (element.requestFullscreen) {
            element.requestFullscreen(options);
        } else if (element.webkitRequestFullscreen) {
            element.webkitRequestFullscreen(options);
        } else if (element.mozRequestFullScreen) {
            element.mozRequestFullScreen(options);
        } else if (element.msRequestFullscreen) {
            element.msRequestFullscreen(options);
        }
    };

    util.exitFullscreen = function() {
        if (document.exitFullscreen) {
            document.exitFullscreen();
        } else if (document.webkitExitFullscreen) {
            document.webkitExitFullscreen();
        } else if (document.mozCancelFullScreen) {
            document.mozCancelFullScreen();
        } else if (document.msExitFullscreen) {
            document.msExitFullscreen();
        }
    };

    util.fullscreenElement = function() {
        return (document.fullscreenElement ||
            document.webkitFullscreenElement ||
            document.mozFullScreenElement ||
            document.msFullscreenElement ||
            null);
    };

    util.addFullscreenchangeEventListener = function(listener) {
        if ("onfullscreenchange" in document) {
            document.addEventListener("fullscreenchange", listener);
        } else if ("onwebkitfullscreenchange" in document) {
            document.addEventListener("webkitfullscreenchange", listener);
        }
    };

    util.requestPointerLock = function(element) {
        if (element.requestPointerLock) {
            element.requestPointerLock();
        } else if (element.mozRequestPointerLock) {
            element.mozRequestPointerLock();
        }
    };

    util.exitPointerLock = function() {
        if (document.exitPointerLock) {
            document.exitPointerLock();
        } else if (document.mozExitPointerLock) {
            document.mozExitPointerLock();
        }
    };

    util.pointerLockElement = function() {
        return (document.pointerLockElement ||
            document.mozPointerLockElement ||
            null);
    };

    util.addPointerlockchangeEventListener = function(listener) {
        if ("onpointerlockchange" in document) {
            document.addEventListener("pointerlockchange", listener);
        } else if ("onmozpointerlockchange" in document) {
            document.addEventListener("mozpointerlockchange", listener);
        }
    };

    return util;
})();

var controller = (function() {
    var controller = {};

    var moveXSum = 0;
    var moveYSum = 0;
    var scrollHSum = 0;
    var scrollVSum = 0;
    var scrolling = false;
    var scrollFinish = false;
    var updateTimoueActive = false;

    function startUpdate(fromTimeout) {
        if (updateTimoueActive && !fromTimeout) {
            return;
        }
        updateTimoueActive = false;
        var finished = true;
        var xInt = Math.trunc(moveXSum);
        var yInt = Math.trunc(moveYSum);
        if (xInt != 0 || yInt != 0) {
            ws.send("m" + xInt + ";" + yInt);
            moveXSum -= xInt;
            moveYSum -= yInt;
            finished = false;
        }
        var hInt = Math.trunc(scrollHSum);
        var vInt = Math.trunc(scrollVSum);
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
        updateTimoueActive = !finished && config.updateRate > 0;
        if (updateTimoueActive) {
            setTimeout(startUpdate, 1000/config.updateRate, true);
        }
    }

    controller.pointerMove = function(deltaX, deltaY) {
        moveXSum += deltaX;
        moveYSum += deltaY;
        startUpdate();
    };

    controller.pointerScroll = function(deltaHorizontal, deltaVertical, finish) {
        scrollHSum += deltaHorizontal;
        scrollVSum += deltaVertical;
        scrollFinish |= finish;
        startUpdate();
    };

    controller.pointerButton = function(button, press) {
        ws.send("b" + button + ";" + (press ? 1 : 0));
    };


    controller.keyboardKey = function(key) {
        ws.send("k" + key);
    };

    controller.keyboardText = function(text) {
        ws.send("t" + text);
    };

    return controller;
})();

var touchpad = (function() {
    var touchpad = {};

    var moved = false;
    var startTimeStamp = 0;
    var lastEndTimeStamp = 0;
    var releasedCount = 0;
    var ongoingTouches = [];
    var dragging = false;
    var draggingTimeout = null;

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

    function calculateAccelerationMult(speed) {
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
        controller.pointerButton(POINTER_BUTTON_LEFT, false);
    }

    touchpad.handleTouchstart = function(evt) {
        // Might get called multiple times for the same touches
        if (ongoingTouches.length == 0) {
            startTimeStamp = evt.timeStamp;
            moved = false;
        }
        var touches = evt.changedTouches;
        var foundTouch = false;
        for (var i = 0; i < touches.length; i += 1) {
            if (ongoingTouches.length == 0 && !touches[i].target.classList.contains("touch")) {
                continue;
            }
            foundTouch = true;
            var touch = copyTouch(touches[i], evt.timeStamp);
            var idx = ongoingTouchIndexById(touch.identifier);
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

    touchpad.handleTouchend = touchpad.handleTouchcancel = function(evt) {
        var touches = evt.changedTouches;
        var foundTouch = false;
        for (var i = 0; i < touches.length; i += 1) {
            var idx = ongoingTouchIndexById(touches[i].identifier);
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
                var button = 0;
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

    touchpad.handleTouchmove = function(evt) {
        var sumX = 0;
        var sumY = 0;
        var touches = evt.changedTouches;
        var foundTouch = false;
        for (var i = 0; i < touches.length; i += 1) {
            var idx = ongoingTouchIndexById(touches[i].identifier);
            if (idx < 0) {
                continue;
            }
            foundTouch = true;
            if (!moved) {
                var dist = Math.sqrt(Math.pow(touches[i].pageX - ongoingTouches[idx].pageXStart, 2) +
                    Math.pow(touches[i].pageY - ongoingTouches[idx].pageYStart, 2));
                if (ongoingTouches.length > TOUCH_MOVE_THRESHOLD.length ||
                    dist > TOUCH_MOVE_THRESHOLD[ongoingTouches.length - 1] ||
                    evt.timeStamp - startTimeStamp >= TOUCH_TIMEOUT) {
                    moved = true;
                }
            }
            var dx = touches[i].pageX - ongoingTouches[idx].pageX;
            var dy = touches[i].pageY - ongoingTouches[idx].pageY;
            var timeDelta = evt.timeStamp - ongoingTouches[idx].timeStamp;
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

var keyboard = (function() {
    var keyboard = {};

    keyboard.handleKeydown = function(evt) {
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
                controller.keyboardKey(key);
            }
        } else if (evt.key.length == 1) {
            evt.preventDefault();
            controller.keyboardText(evt.key);
        }
    };

    return keyboard;
})();

var mouse = (function() {
    var mouse = {};

    var buttons = 0;

    function updateButtons(newButtons) {
        for (var button = 0; button < 3; button += 1) {
            var flag = 1 << button;
            if ((newButtons&flag) != (buttons&flag)) {
                controller.pointerButton(button, newButtons&flag);
            }
        }
        buttons = newButtons;
    }

    mouse.handleMousedown = mouse.handleMouseup = function(evt) {
        updateButtons(evt.buttons);
    };

    mouse.handleMousemove = function(evt) {
        controller.pointerMove(evt.movementX*config.mouseMoveSpeed, evt.movementY*config.mouseMoveSpeed);
    };

    mouse.handleWheel = function(evt) {
        if (evt.deltaMode == WheelEvent.DOM_DELTA_PIXEL) {
            controller.pointerScroll(evt.deltaX*config.mouseScrollSpeed, evt.deltaY*config.mouseScrollSpeed, true);
        } else if (evt.deltaMode == WheelEvent.DOM_DELTA_LINE) {
            controller.pointerScroll(evt.deltaX*20*config.mouseScrollSpeed, evt.deltaY*20*config.mouseScrollSpeed, true);
        }
    };

    return mouse;
})();

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
    var mouseScene = document.getElementById("mouse");
    var activeScene = null;
    var keysActiveName = "";

    function showScene(scene) {
        activeScene = scene;
        if (util.fullscreenElement() && !scene.classList.contains("fullscreen")) {
            util.exitFullscreen();
        }
        if (util.pointerLockElement() && activeScene != mouseScene) {
            util.exitPointerLock();
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
        } else if (util.pointerLockElement()) {
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
    util.addFullscreenchangeEventListener(function() {
        updateUI();
    });
    if (!util.fullscreenEnabled()) {
        fullscreenbutton.classList.add("hidden");
    }
    fullscreenbutton.addEventListener("click", function() {
        if (util.fullscreenElement()) {
            util.exitFullscreen();
        } else {
            util.requestFullscreen(document.documentElement, {navigationUI: "hide"});
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
            controller.keyboardKey(o.key);
        });
    });
    document.getElementById("keyboardkeysbutton").addEventListener("click", function() {
        showKeys("keyboard");
    });
    document.getElementById("sendbutton").addEventListener("click", function() {
        if (keyboardTextarea.value) {
            // normalize line endings
            controller.keyboardText(keyboardTextarea.value.replace(/\r\n?/g, "\n"));
            keyboardTextarea.value = "";
            keyboardTextarea.oninput();
        }
        history.back();
    });
    window.addEventListener("popstate", function() {
        updateUI();
    });
    document.getElementById("reloadbutton").addEventListener("click", function() {
        location.reload();
    });
    document.querySelectorAll(".backbutton").forEach(function(button) {
        button.addEventListener("click", function() {
            history.back();
        });
    });
    document.addEventListener("touchstart", touchpad.handleTouchstart);
    document.addEventListener("touchend", touchpad.handleTouchend);
    document.addEventListener("touchcancel", touchpad.handleTouchcancel);
    document.addEventListener("touchmove", touchpad.handleTouchmove);
    document.addEventListener("keydown", function(evt) {
        if (activeScene && activeScene.classList.contains("key")) {
            keyboard.handleKeydown(evt);
        }
    });
    util.addPointerlockchangeEventListener(function() {
        updateUI();
    });
    document.addEventListener("mousedown", function(event) {
        if (activeScene != mouseScene && event.buttons == 1 && event.target.classList.contains("touch")) {
            util.requestPointerLock(mouseScene);
        }
    });
    ["touchstart", "touchend", "touchcancel", "touchmove"].forEach(function(type) {
        mouseScene.addEventListener(type, function(evt) {
            evt.preventDefault();
        });
    });
    mouseScene.addEventListener("mousedown", mouse.handleMousedown);
    mouseScene.addEventListener("mouseup", mouse.handleMouseup);
    mouseScene.addEventListener("mousemove", mouse.handleMousemove);
    mouseScene.addEventListener("wheel", mouse.handleWheel);
});
